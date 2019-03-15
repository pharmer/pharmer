package gce

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/appscode/go/context"
	"github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	proconfig "github.com/pharmer/pharmer/apis/v1beta1/gce"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	rupdate "google.golang.org/api/replicapoolupdater/v1beta1"
	gcs "google.golang.org/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	ProviderName                 = "gce"
	TemplateURI                  = "https://www.googleapis.com/compute/v1/projects/"
	firewallRuleAnnotationPrefix = "gce.clusterapi.k8s.io/firewall"
	firewallRuleInternalSuffix   = "-allow-cluster-internal"
	firewallRuleApiSuffix        = "-allow-api-public"

	ProjectAnnotationKey = "gcp-project"
	ZoneAnnotationKey    = "gcp-zone"
	NameAnnotationKey    = "gcp-name"
)

var providerIdRE = regexp.MustCompile(`^` + ProviderName + `://([^/]+)/([^/]+)/([^/]+)$`)
var templateNameRE = regexp.MustCompile(`^` + TemplateURI + `([^/]+)/global/instanceTemplates/([^/]+)$`)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	namer   namer

	computeService *compute.Service
	storageService *gcs.Service
	updateService  *rupdate.Service

	owner string
}

func NewConnector(ctx context.Context, cluster *api.Cluster, owner string) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.ClusterConfig().CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.Config.CredentialName)
	}

	cluster.Spec.Config.Cloud.Project = typed.ProjectID()
	conf, err := google.JWTConfigFromJSON([]byte(typed.ServiceAccount()),
		compute.ComputeScope,
		compute.DevstorageReadWriteScope,
		rupdate.ReplicapoolScope)
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}
	client := conf.Client(context.Background())
	computeService, err := compute.New(client)
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}
	storageService, err := gcs.New(client)
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}
	updateService, err := rupdate.New(client)
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}

	if err = proconfig.SetGCEClusterProviderSpec(cluster.Spec.ClusterAPI, cluster.ClusterConfig()); err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}

	conn := cloudConnector{
		ctx:            ctx,
		cluster:        cluster,
		computeService: computeService,
		storageService: storageService,
		updateService:  updateService,
		owner:          owner,
	}
	if ok, msg := conn.IsUnauthorized(typed.ProjectID()); !ok {
		return nil, errors.Errorf("Credential %s does not have necessary authorization. Reason: %s.", cluster.Spec.Config.CredentialName, msg)
	}
	return &conn, nil
}

// Returns true if unauthorized
func (conn *cloudConnector) IsUnauthorized(project string) (bool, string) {
	_, err := conn.computeService.InstanceGroups.List(project, "us-central1-b").Do()
	if err != nil {
		return false, "Credential missing required authorization"
	}
	return true, ""
}

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	return nil
}

func PrepareCloud(ctx context.Context, clusterName string, owner string) (*cloudConnector, error) {
	var err error
	var conn *cloudConnector
	cluster, err := Store(ctx).Owner(owner).Clusters().Get(clusterName)
	if err != nil {
		return conn, fmt.Errorf("cluster `%s` does not exist. Reason: %v", clusterName, err)
	}
	//cm.cluster = cluster

	if ctx, err = LoadCACertificates(ctx, cluster, owner); err != nil {
		return conn, err
	}
	/*if cm.ctx, err = LoadEtcdCertificate(cm.ctx, cm.cluster); err != nil {
		return err
	}*/
	if ctx, err = LoadSSHKey(ctx, cluster, owner); err != nil {
		return conn, err
	}
	/*if cm.ctx, err = LoadSaKey(cm.ctx, cm.cluster); err != nil {
		return err
	}*/

	if conn, err = NewConnector(ctx, cluster, owner); err != nil {
		return nil, err
	}
	conn.namer = namer{cluster: cluster}
	return conn, nil
}

func (conn *cloudConnector) deleteInstance(name string) error {
	Logger(conn.ctx).Info("Deleting instance...")
	r, err := conn.computeService.Instances.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, name).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	err = conn.waitForZoneOperation(r.Name)
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) waitForGlobalOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.GlobalOperations.Get(conn.cluster.Spec.Config.Cloud.Project, operation).Do()
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Operation %v is %v ...", attempt, operation, r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) waitForRegionOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.RegionOperations.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Region, operation).Do()
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Operation %v is %v ...", attempt, operation)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) waitForZoneOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.ZoneOperations.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, operation).Do()
		if err != nil {
			return false, nil
		}
		Logger(conn.ctx).Infof("Attempt %v: Operation %v is %v ...", attempt, operation, r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) importPublicKey() error {
	Logger(conn.ctx).Infof("Importing SSH key with fingerprint: %v", SSHKey(conn.ctx).OpensshFingerprint)
	pubKey := string(SSHKey(conn.ctx).PublicKey)
	r1, err := conn.computeService.Projects.SetCommonInstanceMetadata(conn.cluster.Spec.Config.Cloud.Project, &compute.Metadata{
		Items: []*compute.MetadataItems{
			{
				Key:   conn.cluster.Spec.Config.Cloud.SSHKeyName,
				Value: &pubKey,
			},
		},
	}).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}

	err = conn.waitForGlobalOperation(r1.Name)
	if err != nil {
		errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Debug("Imported SSH key")
	Logger(conn.ctx).Info("SSH key imported")
	return nil
}

func (conn *cloudConnector) getNetworks() (bool, error) {
	Logger(conn.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, conn.cluster.Spec.Config.Cloud.Project)
	r1, err := conn.computeService.Networks.Get(conn.cluster.Spec.Config.Cloud.Project, defaultNetwork).Do()
	Logger(conn.ctx).Debug("Retrieve network result", r1, err)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) ensureNetworks() error {
	Logger(conn.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, conn.cluster.Spec.Config.Cloud.Project)
	r2, err := conn.computeService.Networks.Insert(conn.cluster.Spec.Config.Cloud.Project, &compute.Network{
		IPv4Range: "10.240.0.0/16",
		Name:      defaultNetwork,
	}).Do()
	Logger(conn.ctx).Debug("Created new network", r2, err)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("New network %v is created", defaultNetwork)

	return nil
}

func (conn *cloudConnector) getFirewallRules() (bool, error) {
	ruleClusterInternal := conn.cluster.Name + firewallRuleInternalSuffix
	Logger(conn.ctx).Infof("Retrieving firewall rule %v", ruleClusterInternal)
	if r1, err := conn.computeService.Firewalls.Get(conn.cluster.Spec.Config.Cloud.Project, ruleClusterInternal).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved firewall rule", r1, err)
		return false, err
	}

	ruleSSH := conn.cluster.Name + "-allow-ssh"
	if r2, err := conn.computeService.Firewalls.Get(conn.cluster.Spec.Config.Cloud.Project, ruleSSH).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved firewall rule", r2, err)
		return false, err
	}

	ruleApiPublic := conn.cluster.Name + firewallRuleApiSuffix
	if r5, err := conn.computeService.Firewalls.Get(conn.cluster.Spec.Config.Cloud.Project, ruleApiPublic).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved firewall rule", r5, err)
		return false, err
	}

	return true, nil
}

func (conn *cloudConnector) ensureFirewallRules() error {
	network := fmt.Sprintf("projects/%v/global/networks/%v", conn.cluster.Spec.Config.Cloud.Project, defaultNetwork)
	var err error
	cluster := conn.cluster
	ruleInternal := defaultNetwork + "-allow-internal"
	if r1, err := conn.computeService.Firewalls.Get(conn.cluster.Spec.Config.Cloud.Project, ruleInternal).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved firewall rule", r1, err)

		r2, err := conn.computeService.Firewalls.Insert(conn.cluster.Spec.Config.Cloud.Project, &compute.Firewall{
			Name:         ruleInternal,
			Network:      network,
			SourceRanges: []string{"10.128.0.0/9"}, // 10.0.0.0/8
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
					Ports:      []string{"0-65535"},
				},
				{
					IPProtocol: "udp",
					Ports:      []string{"0-65535"},
				},
				{
					IPProtocol: "icmp",
				},
			},
		}).Do()
		Logger(conn.ctx).Debug("Created firewall rule", r2, err)
		if err != nil {
			return errors.Wrap(err, ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Firewall rule %v created", ruleInternal)
	}

	ruleClusterInternal := cluster.Name + firewallRuleInternalSuffix

	if r1, err := conn.computeService.Firewalls.Get(cluster.Spec.Config.Cloud.Project, ruleClusterInternal).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved firewall rule", r1, err)

		r2, err := conn.computeService.Firewalls.Insert(cluster.Spec.Config.Cloud.Project, &compute.Firewall{
			Name:    ruleClusterInternal,
			Network: network,
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
				},
			},
			TargetTags: []string{conn.namer.cluster.Name + "-worker"},
			SourceTags: []string{conn.namer.cluster.Name + "-worker"},
		}).Do()

		Logger(conn.ctx).Debug("Created firewall rule", r2, err)
		if err != nil {
			return errors.Wrap(err, ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Firewall rule %v created", ruleClusterInternal)
	}
	if cluster.Spec.ClusterAPI.ObjectMeta.Annotations == nil {
		cluster.Spec.ClusterAPI.ObjectMeta.Annotations = make(map[string]string)
	}
	cluster.Spec.ClusterAPI.ObjectMeta.Annotations[firewallRuleAnnotationPrefix+ruleClusterInternal] = "true"
	if conn.cluster, err = Store(conn.ctx).Owner(conn.owner).Clusters().Update(cluster); err != nil {
		return err
	}

	ruleSSH := cluster.Name + "-allow-ssh"
	if r3, err := conn.computeService.Firewalls.Get(conn.cluster.Spec.Config.Cloud.Project, ruleSSH).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved firewall rule", r3, err)

		r4, err := conn.computeService.Firewalls.Insert(conn.cluster.Spec.Config.Cloud.Project, &compute.Firewall{
			Name:         ruleSSH,
			Network:      network,
			SourceRanges: []string{"0.0.0.0/0"},
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
					Ports:      []string{"22"},
				},
			},
		}).Do()
		Logger(conn.ctx).Debug("Created firewall rule", r4, err)
		if err != nil {
			return errors.Wrap(err, ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Firewall rule %v created", ruleSSH)
	}

	ruleApiPublic := cluster.Name + firewallRuleApiSuffix

	if r7, err := conn.computeService.Firewalls.Get(conn.cluster.Spec.Config.Cloud.Project, ruleApiPublic).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved firewall rule", r7, err)
		r8, err := conn.computeService.Firewalls.Insert(conn.cluster.Spec.Config.Cloud.Project, &compute.Firewall{
			Name:    ruleApiPublic,
			Network: "global/networks/default",
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
					Ports:      []string{"6443"},
				},
				{
					IPProtocol: "tcp",
					Ports:      []string{"443"},
				},
			},
			TargetTags:   []string{"https-server"},
			SourceRanges: []string{"0.0.0.0/0"},
		}).Do()

		Logger(conn.ctx).Debug("Created firewall rule", r8, err)

		if err != nil {
			return errors.Wrap(err, ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Firewall rule %v created", ruleSSH)
	}

	cluster.Spec.ClusterAPI.ObjectMeta.Annotations[firewallRuleAnnotationPrefix+ruleApiPublic] = "true"
	if conn.cluster, err = Store(conn.ctx).Owner(conn.owner).Clusters().Update(cluster); err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) getMasterPDDisk(name string) (bool, error) {

	if r, err := conn.computeService.Disks.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, name).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved master persistent disk", r, err)
		return false, err
	}

	//conn.cluster.Spec.MasterDiskId = name
	return true, nil
}

func (conn *cloudConnector) createDisk(name, diskType string, sizeGb int64) (string, error) {
	// Type:        "https://www.googleapis.com/compute/v1/projects/tigerworks-kube/zones/us-central1-b/diskTypes/pd-ssd",
	// SourceImage: "https://www.googleapis.com/compute/v1/projects/google-containers/global/images/container-vm-v20150806",

	dType := fmt.Sprintf("projects/%v/zones/%v/diskTypes/%v", conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, diskType)

	r1, err := conn.computeService.Disks.Insert(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, &compute.Disk{
		Name:   name,
		Zone:   conn.cluster.Spec.Config.Cloud.Zone,
		Type:   dType,
		SizeGb: sizeGb,
	}).Do()

	Logger(conn.ctx).Debug("Created master disk", r1, err)
	if err != nil {
		return name, errors.Wrap(err, ID(conn.ctx))
	}
	err = conn.waitForZoneOperation(r1.Name)
	if err != nil {
		return name, errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Blank disk of type %v created before creating the master VM", dType)
	return name, nil
}

func (conn *cloudConnector) getReserveIP() (bool, error) {
	/*Logger(conn.ctx).Infof("Checking existence of reserved master ip ")
	if conn.cluster.Spec.MasterReservedIP == "auto" {
		name := conn.namer.ReserveIPName()

		Logger(conn.ctx).Infof("Checking existence of reserved master ip %v", name)
		if r1, err := conn.computeService.Addresses.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Region, name).Do(); err == nil {
			if r1.Status == "IN_USE" {
				return true, errors.Errorf("Found a static IP with name %v in use. Failed to reserve a new ip with the same name.", name)
			}

			Logger(conn.ctx).Debug("Found master IP was already reserved", r1, err)
			conn.cluster.Spec.MasterReservedIP = r1.Address
			Logger(conn.ctx).Infof("Newly reserved master ip %v", conn.cluster.Spec.MasterReservedIP)
			return true, nil
		}
	} */
	return false, nil

}

func (conn *cloudConnector) reserveIP() (string, error) {
	name := conn.namer.ReserveIPName()
	if _, err := conn.getReserveIP(); err != nil {
		return "", err
	}

	Logger(conn.ctx).Infof("Reserving master ip %v", name)
	r2, err := conn.computeService.Addresses.Insert(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Region, &compute.Address{Name: name}).Do()
	Logger(conn.ctx).Debug("Reserved master IP", r2, err)
	if err != nil {
		return "", errors.Wrap(err, ID(conn.ctx))
	}

	err = conn.waitForRegionOperation(r2.Name)
	if err != nil {
		return "", errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Master ip %v reserved", name)
	if r3, err := conn.computeService.Addresses.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Region, name).Do(); err == nil {
		Logger(conn.ctx).Debug("Retrieved newly reserved master IP", r3, err)
		return r3.Name, nil
	}

	return "", nil
}

func (conn *cloudConnector) getMasterInstance() (bool, error) {
	if r1, err := conn.computeService.Instances.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, conn.namer.MasterName()).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved master instance", r1, err)
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) createMasterIntance(cluster *api.Cluster, machine *clusterv1.Machine) (string, error) {
	// MachineType:  "projects/tigerworks-kube/zones/us-central1-b/machineTypes/n1-standard-1",
	// Zone:         "projects/tigerworks-kube/zones/us-central1-b",

	script, err := conn.renderStartupScript(cluster, machine, "")
	if err != nil {
		return "", err
	}

	if found, _ := conn.getMasterPDDisk(conn.namer.MasterPDName()); !found {
		_, err := conn.createDisk(conn.namer.MasterPDName(), "pd-standard", 30)
		if err != nil {
			return "", err
		}
	}

	providerSpec := proconfig.GetGCEMachineProviderSpec(machine.Spec.ProviderSpec)

	machineType := fmt.Sprintf("projects/%v/zones/%v/machineTypes/%v", conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, providerSpec.MachineType)
	zone := fmt.Sprintf("projects/%v/zones/%v", conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone)
	pdSrc := fmt.Sprintf("projects/%v/zones/%v/disks/%v", conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, conn.namer.MasterPDName())
	srcImage := fmt.Sprintf("projects/%v/global/images/%v", conn.cluster.Spec.Config.Cloud.InstanceImageProject, conn.cluster.Spec.Config.Cloud.InstanceImage)

	pubKey := string(SSHKey(conn.ctx).PublicKey)
	value := fmt.Sprintf("%v:%v %v", conn.namer.AdminUsername(), pubKey, conn.namer.AdminUsername())

	instance := &compute.Instance{
		Name:        conn.namer.MasterName(),
		Zone:        zone,
		MachineType: machineType,
		// --image-project="${MASTER_IMAGE_PROJECT}"
		// --image "${MASTER_IMAGE}"
		Tags: &compute.Tags{
			Items: []string{
				"https-server",
				conn.cluster.Name + "-master",
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Network: fmt.Sprintf("projects/%v/global/networks/%v", conn.cluster.Spec.Config.Cloud.Project, defaultNetwork),
				AccessConfigs: []*compute.AccessConfig{
					{
						Name: "Master External IP",
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Scopes: []string{
					"https://www.googleapis.com/auth/compute",
					"https://www.googleapis.com/auth/devstorage.read_only",
					"https://www.googleapis.com/auth/logging.write",
				},
				Email: "default",
			},
		},
		CanIpForward: true,
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{
					Key:   "startup-script",
					Value: types.StringP(script),
				},
				{
					Key:   "ssh-keys",
					Value: &value,
				},
			},
		},
		/*
		  "disks": [
		    {
		      "kind": "compute#attachedDisk",
		      "type": "PERSISTENT",
		      "mode": "READ_WRITE",
		      "source": "projects/tigerworks-kube/zones/us-central1-b/disks/kubernetes-master",
		      "deviceName": "persistent-disk-0",
		      "index": 0,
		      "boot": true,
		      "autoDelete": true,
		      "interface": "SCSI"
		    },
		    {
		      "kind": "compute#attachedDisk",
		      "type": "PERSISTENT",
		      "mode": "READ_WRITE",
		      "source": "projects/tigerworks-kube/zones/us-central1-b/disks/kubernetes-master-pd",
		      "deviceName": "master-pd",
		      "index": 1,
		      "boot": false,
		      "autoDelete": false,
		      "interface": "SCSI"
		    }
		  ],
		*/
		Disks: []*compute.AttachedDisk{
			{
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: srcImage,
				},
				Mode:       "READ_WRITE",
				Boot:       true,
				AutoDelete: true,
			},
			{
				DeviceName: "master-pd",
				Mode:       "READ_WRITE",
				Boot:       false,
				AutoDelete: false,
				Source:     pdSrc,
			},
		},
	}

	/*
		if ng.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
			instance.NetworkInterfaces[0].AccessConfigs = []*compute.AccessConfig{
				{
					NatIP: conn.cluster.Status.ReservedIPs[0].IP,
				},
			}
		} else {
			instance.NetworkInterfaces[0].AccessConfigs = []*compute.AccessConfig{
				{
					Name: "Master External IP",
					Type: "ONE_TO_ONE_NAT",
				},
			}
		}
	*/
	r1, err := conn.computeService.Instances.Insert(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, instance).Do()
	Logger(conn.ctx).Debug("Created master instance", r1, err)

	if err != nil {
		return r1.Name, errors.Wrap(err, ID(conn.ctx))
	}
	//Logger(conn.ctx).Infof("Master instance of type %v in zone %v using persistent disk %v created", machineType, zone, pdSrc)
	Logger(conn.ctx).Infof("Master instance of type %v in zone %v is created", machineType, zone)
	log.Println(r1.Name)

	return r1.Name, nil
}

// Instance
func (conn *cloudConnector) getInstance(instance string) (*api.NodeInfo, error) {
	Logger(conn.ctx).Infof("Retrieving instance %v in zone %v", instance, conn.cluster.Spec.Config.Cloud.Zone)
	r1, err := conn.computeService.Instances.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, instance).Do()
	Logger(conn.ctx).Debug("Retrieved instance", r1, err)
	if err != nil {
		return nil, err
	}
	return conn.newKubeInstance(r1)
}

func (conn *cloudConnector) listInstances(instanceGroup string) ([]*api.NodeInfo, error) {
	Logger(conn.ctx).Infof("Retrieving instances in node group %v", instanceGroup)
	instances := make([]*api.NodeInfo, 0)
	r1, err := conn.computeService.InstanceGroups.ListInstances(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, instanceGroup, &compute.InstanceGroupsListInstancesRequest{
		InstanceState: "ALL",
	}).Do()
	Logger(conn.ctx).Debug("Retrieved instance", r1, err)
	if err != nil {
		return nil, errors.Wrap(err, ID(conn.ctx))
	}

	for _, item := range r1.Items {
		name := item.Instance[strings.LastIndex(item.Instance, "/")+1:]
		r2, err := conn.computeService.Instances.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, name).Do()
		if err != nil {
			return nil, errors.Wrap(err, ID(conn.ctx))
		}
		instance, err := conn.newKubeInstance(r2)
		if err != nil {
			return nil, errors.Wrap(err, ID(conn.ctx))
		}
		//		instance.Spec.Role = api.RoleNode
		instances = append(instances, instance)
	}
	return instances, nil
}

func (conn *cloudConnector) newKubeInstance(r1 *compute.Instance) (*api.NodeInfo, error) {
	for _, accessConfig := range r1.NetworkInterfaces[0].AccessConfigs {
		if accessConfig.Type == "ONE_TO_ONE_NAT" {
			i := api.NodeInfo{
				Name:       r1.Name,
				ExternalID: strconv.FormatUint(r1.Id, 10),
				PublicIP:   accessConfig.NatIP,
				PrivateIP:  r1.NetworkInterfaces[0].NetworkIP,
			}

			/*
				// Status: [Output Only] The status of the instance. One of the
				// following values: PROVISIONING, STAGING, RUNNING, STOPPING,
				// SUSPENDED, SUSPENDING, and TERMINATED.
				//
				// Possible values:
				//   "PROVISIONING"
				//   "RUNNING"
				//   "STAGING"
				//   "STOPPED"
				//   "STOPPING"
				//   "SUSPENDED"
				//   "SUSPENDING"
				//   "TERMINATED"
			*/

			return &i, nil
		}
	}
	return nil, errors.Errorf("[%s] failed to convert gcloud instance to KubeInstance.", ID(conn.ctx))
}

func (conn *cloudConnector) getNodeFirewallRule() (bool, error) {
	name := conn.cluster.Name + "-node-all"
	if r1, err := conn.computeService.Firewalls.Get(conn.cluster.Spec.Config.Cloud.Project, name).Do(); err != nil {
		Logger(conn.ctx).Debug("Retrieved node firewall rule", r1, err)
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) createNodeFirewallRule() (string, error) {
	/*
		name := conn.cluster.Name + "-node-all"
		network := fmt.Sprintf("projects/%v/global/networks/%v", conn.cluster.Spec.Config.Cloud.Project, defaultNetwork)

		r1, err := conn.computeService.Firewalls.Insert(conn.cluster.Spec.Config.Cloud.Project, &compute.Firewall{
			Name:         name,
			Network:      network,
			SourceRanges: []string{conn.cluster.Spec.Networking.PodSubnet},
			TargetTags:   []string{conn.cluster.Name + "-node"},
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
				},
				{
					IPProtocol: "udp",
				},
				{
					IPProtocol: "icmp",
				},
				{
					IPProtocol: "esp",
				},
				{
					IPProtocol: "ah",
				},
				{
					IPProtocol: "sctp",
				},
			},
		}).Do()
		Logger(conn.ctx).Debug("Created firewall rule", r1, err)
		if err != nil {
			return "", errors.Wrap(err, ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Node firewall rule %v created", name)
		return r1.Name, nil */

	return "", nil
}

func (conn *cloudConnector) createNodeInstanceTemplate(ng *clusterv1.Machine, token string) (string, error) {
	/*
		templateName := conn.namer.InstanceTemplateName(ng.Spec.Template.Spec.SKU)

		Logger(conn.ctx).Infof("Retrieving node template %v", templateName)
		if r1, err := conn.computeService.InstanceTemplates.Get(conn.cluster.Spec.Config.Cloud.Project, templateName).Do(); err == nil {
			Logger(conn.ctx).Debug("Retrieved node template", r1, err)

			Logger(conn.ctx).Infof("Deleting node template %v", templateName)
			if r2, err := conn.computeService.InstanceTemplates.Delete(conn.cluster.Spec.Config.Cloud.Project, templateName).Do(); err != nil {
				Logger(conn.ctx).Debug("Delete node template called", r2, err)
				Logger(conn.ctx).Infoln("Failed to delete existing instance template")
				return "", err
			}
		}
		//  if cluster.Spec.ctx.Preemptiblenode == "true" {
		//	  preemptible_nodes = "--preemptible --maintenance-policy TERMINATE"
		//  }

		script, err := conn.renderStartupScript(ng, token)
		if err != nil {
			return "", err
		}
		pubKey := string(SSHKey(conn.ctx).PublicKey)
		value := fmt.Sprintf("%v:%v %v", conn.namer.AdminUsername(), pubKey, conn.namer.AdminUsername())
		image := fmt.Sprintf("projects/%v/global/images/%v", conn.cluster.Spec.Config.Cloud.InstanceImageProject, conn.cluster.Spec.Config.Cloud.InstanceImage)
		network := fmt.Sprintf("projects/%v/global/networks/%v", conn.cluster.Spec.Config.Cloud.Project, defaultNetwork)

		Logger(conn.ctx).Infof("Create instance template %v", templateName)
		tpl := &compute.InstanceTemplate{
			Name: templateName,
			Properties: &compute.InstanceProperties{
				MachineType: ng.Spec.Template.Spec.SKU,
				Scheduling: &compute.Scheduling{
					AutomaticRestart:  types.FalseP(),
					OnHostMaintenance: "TERMINATE",
				},
				Disks: []*compute.AttachedDisk{
					{
						AutoDelete: true,
						Boot:       true,
						InitializeParams: &compute.AttachedDiskInitializeParams{
							DiskType:    ng.Spec.Template.Spec.DiskType,
							DiskSizeGb:  ng.Spec.Template.Spec.DiskSize,
							SourceImage: image,
						},
					},
				},
				Tags: &compute.Tags{
					Items: []string{conn.cluster.Name + "-node"},
				},
				NetworkInterfaces: []*compute.NetworkInterface{
					{
						Network: network,
						//AccessConfigs: []*compute.AccessConfig{
						//	{
						//		Name: "External IP",
						//		Type: "ONE_TO_ONE_NAT",
						//	},
						//},
					},
				},
				ServiceAccounts: []*compute.ServiceAccount{
					{
						Scopes: []string{
							"https://www.googleapis.com/auth/compute",
							"https://www.googleapis.com/auth/devstorage.read_only",
							"https://www.googleapis.com/auth/logging.write",
						},
						Email: "default",
					},
				},
				CanIpForward: true,
				Metadata: &compute.Metadata{
					Items: []*compute.MetadataItems{
						{
							Key:   "startup-script",
							Value: types.StringP(script),
						},
						{
							Key:   "ssh-keys",
							Value: &value,
						},
					},
				},
			},
		}
		// if conn.cluster.Spec.EnableNodePublicIP {
		tpl.Properties.NetworkInterfaces[0].AccessConfigs = []*compute.AccessConfig{
			{
				Name: "Node External IP",
				Type: "ONE_TO_ONE_NAT",
			},
		}
		// }
		r1, err := conn.computeService.InstanceTemplates.Insert(conn.cluster.Spec.Config.Cloud.Project, tpl).Do()
		Logger(conn.ctx).Debug("Create instance template called", r1, err)
		if err != nil {
			return "", errors.Wrap(err, ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Node instance template %v created for sku %v", templateName, ng.Spec.Template.Spec.SKU)
		return r1.Name, nil */
	return "", nil
}

func (conn *cloudConnector) createNodeGroup(ng *clusterv1.Machine) (string, error) {
	/*
		template := fmt.Sprintf("projects/%v/global/instanceTemplates/%v", conn.cluster.Spec.Config.Cloud.Project, conn.namer.InstanceTemplateName(ng.Spec.Template.Spec.SKU))

		Logger(conn.ctx).Infof("Creating instance group %v from template %v", ng.Name, template)
		r1, err := conn.computeService.InstanceGroupManagers.Insert(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, &compute.InstanceGroupManager{
			Name:             ng.Name,
			BaseInstanceName: rand.WithUniqSuffix(ng.Name),
			TargetSize:       ng.Spec.Nodes,
			InstanceTemplate: template,
		}).Do()
		Logger(conn.ctx).Debug("Create instance group called", r1, err)
		if err != nil {
			return "", errors.Wrap(err, ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Instance group %v created with %v nodes of %v sku", ng.Name, ng.Spec.Nodes, ng.Spec.Template.Spec.SKU)
		return r1.Name, nil */
	return "", nil
}

// Not used since Cluster 1.3
/*func (conn *cloudConnector) createAutoscaler(ng *clusterv1.Machine) (string, error) {
	target := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%v/zones/%v/instanceGroupManagers/%v", conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, ng.Name)

	Logger(conn.ctx).Infof("Creating auto scaler %v for instance group %v", ng.Name, target)
	r1, err := conn.computeService.Autoscalers.Insert(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, &compute.Autoscaler{
		Name:   ng.Name,
		Target: target,
		AutoscalingPolicy: &compute.AutoscalingPolicy{
			MinNumReplicas: int64(1),
			MaxNumReplicas: ng.Spec.Nodes,
		},
	}).Do()
	Logger(conn.ctx).Debug("Create auto scaler called", r1, err)
	if err != nil {
		return "", errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Auto scaler %v for instance group %v created", ng.Name, target)
	return r1.Name, nil
}*/

func (conn *cloudConnector) deleteOnlyNodeGroup(instanceGroupName, template string) error {
	_, err := conn.computeService.InstanceGroupManagers.ListManagedInstances(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, instanceGroupName).Do()
	if err != nil {
		Logger(conn.ctx).Infof("Error on getting nodegroup list. Reason: %v", err)
		return errors.Wrap(err, ID(conn.ctx))
	}

	r1, err := conn.computeService.InstanceGroupManagers.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, instanceGroupName).Do()
	if err != nil {
		Logger(conn.ctx).Infof("Error on deleting nodegroup. Reason: %v", err)
		return errors.Wrap(err, ID(conn.ctx))
	}
	operation := r1.Name
	conn.waitForZoneOperation(operation)
	Logger(conn.ctx).Infof("Instance group %v is deleted", instanceGroupName)
	Logger(conn.ctx).Infof("Instance template %v is deleting", template)
	if err = conn.deleteInstanceTemplate(template); err != nil {
		Logger(conn.ctx).Infof("Error on deleting instance template. Reason: %v", err)
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Instance template %v is deleted", template)
	return nil
}

//delete template
func (conn *cloudConnector) deleteInstanceTemplate(template string) error {
	op, err := conn.computeService.InstanceTemplates.Delete(conn.cluster.Spec.Config.Cloud.Project, template).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	return conn.waitForGlobalOperation(op.Name)
}

func (conn *cloudConnector) deleteGroupInstances(ng *clusterv1.Machine, instance string) error {
	req := &compute.InstanceGroupManagersDeleteInstancesRequest{
		Instances: []string{
			fmt.Sprintf("zones/%v/instances/%v", conn.cluster.Spec.Config.Cloud.Zone, instance),
		},
	}
	r, err := conn.computeService.InstanceGroupManagers.DeleteInstances(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, ng.Name, req).Do()
	fmt.Println(r, err)
	if err != nil {
		return err
	}
	if err = conn.waitForZoneOperation(r.Name); err != nil {
		return err
	}

	return nil
}

func (conn *cloudConnector) addNodeIntoGroup(ng *clusterv1.Machine, size int64) error {
	_, err := conn.computeService.InstanceGroupManagers.ListManagedInstances(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, ng.Name).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	//sz := int64(len(r.ManagedInstances))
	resp, err := conn.computeService.InstanceGroupManagers.Resize(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, ng.Name, size).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	conn.waitForZoneOperation(resp.Name)
	fmt.Println(resp.Name)
	Logger(conn.ctx).Infof("Instance group %v resized", ng.Name)
	/*err = WaitForReadyNodes(conn.ctx, size-sz)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}*/
	return nil
}

func (conn *cloudConnector) deleteMaster() error {
	r2, err := conn.computeService.Instances.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, conn.namer.MasterName()).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	operation := r2.Name
	err = conn.waitForZoneOperation(operation)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Master instance %v deleted", conn.namer.MasterName())
	return nil
}

//delete disk
func (conn *cloudConnector) deleteDisk(nodeDiskNames []string) error {
	masterDisk := conn.namer.MasterPDName()
	r6, err := conn.computeService.Disks.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, masterDisk).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Debugf("Master Disk response %v", r6)
	time.Sleep(5 * time.Second)
	r7, err := conn.computeService.Disks.List(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	for i := range r7.Items {
		s := strings.Split(r7.Items[i].Name, "-")
		diskName1 := strings.Join(s[0:len(s)-1], "-")
		diskName2 := strings.Join(s[0:len(s)-2], "-")

		if diskName1 == conn.cluster.Name || diskName2 == conn.cluster.Name {
			r, err := conn.computeService.Disks.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, r7.Items[i].Name).Do()
			if err != nil {
				return errors.Wrap(err, ID(conn.ctx))
			}
			Logger(conn.ctx).Infof("Disk %v deleted, response %v", r7.Items[i].Name, r.Status)
			time.Sleep(5 * time.Second)
		}

	}

	for _, diskName := range nodeDiskNames {
		r, err := conn.computeService.Disks.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, diskName).Do()
		if err != nil {
			//return errors.Wrap(err, ID(conn.ctx))
			Logger(conn.ctx).Infoln(err)
		}
		Logger(conn.ctx).Infof("Disk %v deleted, response %v", diskName, r.Status)
		time.Sleep(5 * time.Second)
	}

	return nil
}

func (conn *cloudConnector) deleteRoutes() error {
	r1, err := conn.computeService.Routes.List(conn.cluster.Spec.Config.Cloud.Project).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	for i := range r1.Items {
		routeName := r1.Items[i].Name
		if strings.HasPrefix(routeName, conn.cluster.Name) {
			fmt.Println(routeName)
			r2, err := conn.computeService.Routes.Delete(conn.cluster.Spec.Config.Cloud.Project, routeName).Do()
			if err != nil {
				return errors.Wrap(err, ID(conn.ctx))
			}
			Logger(conn.ctx).Infof("Route %v deleted, response %v", routeName, r2.Status)
		}
	}
	return nil
}

//delete firewalls
// ruleClusterInternal := cluster.Name + firewallRuleInternalSuffix
// ruleSSH := cluster.Name + "-allow-ssh"
// ruleApiPublic := cluster.Name + firewallRuleApiSuffix

func (conn *cloudConnector) deleteFirewalls() error {
	time.Sleep(3 * time.Second)

	ruleClusterInternal := conn.cluster.Name + firewallRuleInternalSuffix
	r2, err := conn.computeService.Firewalls.Delete(conn.cluster.Spec.Config.Cloud.Project, ruleClusterInternal).Do()
	if err != nil {
		Logger(conn.ctx).Infoln(err)
		//return errors.Wrap(err, ID(conn.ctx))
	} else {
		Logger(conn.ctx).Infof("Firewalls %v deleted, response %v", ruleClusterInternal, r2.Status)
	}
	time.Sleep(3 * time.Second)

	ruleSSH := conn.cluster.Name + "-allow-ssh"
	r3, err := conn.computeService.Firewalls.Delete(conn.cluster.Spec.Config.Cloud.Project, ruleSSH).Do()
	if err != nil {
		Logger(conn.ctx).Infoln(err)
		//return errors.Wrap(err, ID(conn.ctx))
	} else {
		Logger(conn.ctx).Infof("Firewalls %v deleted, response %v", ruleSSH, r3.Status)
	}
	time.Sleep(3 * time.Second)

	ruleApiPublic := conn.cluster.Name + firewallRuleApiSuffix
	r4, err := conn.computeService.Firewalls.Delete(conn.cluster.Spec.Config.Cloud.Project, ruleApiPublic).Do()
	if err != nil {
		Logger(conn.ctx).Infoln(err)
		//return errors.Wrap(err, ID(conn.ctx))
	} else {
		Logger(conn.ctx).Infof("Firewalls %v deleted, response %v", ruleApiPublic, r4.Status)
	}
	time.Sleep(3 * time.Second)
	return nil
}

// delete reserve ip
func (conn *cloudConnector) releaseReservedIP() error {
	name := conn.namer.ReserveIPName()
	r1, err := conn.computeService.Addresses.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Region, name).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Releasing reserved master ip %v", r1.Address)
	r2, err := conn.computeService.Addresses.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Region, name).Do()
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	err = conn.waitForRegionOperation(r2.Name)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("Master ip %v released", r1.Address)
	return nil
}

func (conn *cloudConnector) getExistingInstanceTemplate(ng *clusterv1.Machine) (string, error) {
	ig, err := conn.computeService.InstanceGroupManagers.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, ng.Name).Do()
	if err != nil {
		return "", err
	}
	//"instanceTemplate": "https://www.googleapis.com/compute/v1/projects/k8s-qa/global/instanceTemplates/gc1-n1-standard-2-v1508392105708944214",
	matches := templateNameRE.FindStringSubmatch(ig.InstanceTemplate)
	if len(matches) != 3 {
		return "", errors.New("error splitting providerID")
	}
	return matches[2], nil
}

//Node template update
func (conn *cloudConnector) updateNodeGroupTemplate(ng *clusterv1.Machine, token string) error {
	op, err := conn.createNodeInstanceTemplate(ng, token)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	err = conn.waitForGlobalOperation(op)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	oldInstanceTemplate, err := conn.getExistingInstanceTemplate(ng)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}

	newInstanceTemplate := &compute.InstanceGroupManagersSetInstanceTemplateRequest{
		//InstanceTemplate: fmt.Sprintf("projects/%v/global/instanceTemplates/%v", conn.cluster.Spec.Config.Cloud.Project, conn.namer.InstanceTemplateName(ng.Spec.Template.Spec.SKU)),
	}
	op2, err := conn.computeService.InstanceGroupManagers.SetInstanceTemplate(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, ng.Name, newInstanceTemplate).Do()
	if err != nil {
		return err
	}
	if err = conn.waitForZoneOperation(op2.Name); err != nil {
		return err
	}

	err = conn.deleteInstanceTemplate(oldInstanceTemplate)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}

	return nil
}

// splitProviderID splits a provider's id into core components.
// A providerID is build out of '${ProviderName}://${project-id}/${zone}/${instance-name}'
// ref: https://github.com/kubernetes/kubernetes/blob/0b9efaeb34a2fc51ff8e4d34ad9bc6375459c4a4/pkg/cloudprovider/providers/gce/gce_util.go#L156
func splitProviderID(providerID string) (project, zone, instance string, err error) {
	matches := providerIdRE.FindStringSubmatch(providerID)
	if len(matches) != 4 {
		return "", "", "", errors.New("error splitting providerID")
	}
	return matches[1], matches[2], matches[3], nil
}
