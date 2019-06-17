package gce

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapiGCE "github.com/pharmer/pharmer/apis/v1beta1/gce"
	proconfig "github.com/pharmer/pharmer/apis/v1beta1/gce"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
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
)

var providerIdRE = regexp.MustCompile(`^` + ProviderName + `://([^/]+)/([^/]+)/([^/]+)$`)

type cloudConnector struct {
	*CloudManager

	namer          namer
	computeService *compute.Service
	storageService *gcs.Service
	updateService  *rupdate.Service
}

func NewConnector(cm *ClusterManager) (*cloudConnector, error) {
	cred, err := store.StoreProvider.Credentials().Get(cm.Cluster.ClusterConfig().CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cm.Cluster.Spec.Config.CredentialName)
	}

	cm.Cluster.Spec.Config.Cloud.Project = typed.ProjectID()
	conf, err := google.JWTConfigFromJSON([]byte(typed.ServiceAccount()),
		compute.ComputeScope,
		compute.DevstorageReadWriteScope,
		rupdate.ReplicapoolScope)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	client := conf.Client(context.Background())
	computeService, err := compute.New(client)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	storageService, err := gcs.New(client)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	updateService, err := rupdate.New(client)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	clusterConfig, err := clusterapiGCE.ClusterConfigFromProviderSpec(cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return nil, errors.Wrap(err, "Error decoding cluster specs from provider config")
	}
	clusterConfig.Project = cm.Cluster.Spec.Config.Cloud.Project

	rawSpec, err := clusterapiGCE.EncodeClusterSpec(clusterConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to encode cluster spec")
	}
	cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	conn := cloudConnector{
		CloudManager:   cm.CloudManager,
		namer:          namer{cm.Cluster},
		computeService: computeService,
		storageService: storageService,
		updateService:  updateService,
	}
	if ok, msg := conn.IsUnauthorized(typed.ProjectID()); !ok {
		return nil, errors.Errorf("Credential %s does not have necessary authorization. Reason: %s.", cm.Cluster.Spec.Config.CredentialName, msg)
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

func (cm *ClusterManager) GetCloudConnector() error {
	var err error

	if cm.conn, err = NewConnector(cm); err != nil {
		return err
	}

	return nil
}

func errAlreadyExists(err error) bool {
	return strings.Contains(err.Error(), "alreadyExists")
}

func (conn *cloudConnector) getLoadBalancer() (string, error) {
	log.Infof("Getting Load Balancer for cluster %s", conn.Cluster.Name)

	project := conn.Cluster.Spec.Config.Cloud.Project
	region := conn.Cluster.Spec.Config.Cloud.Region
	clusterName := conn.Cluster.Name
	name := fmt.Sprintf("%s-apiserver", clusterName)

	address, err := conn.computeService.Addresses.Get(project, region, name).Do()
	if err != nil {
		return "", errors.Wrap(err, "Error getting load balancer ip address")
	}

	if _, err := conn.computeService.HttpHealthChecks.Get(project, name).Do(); err != nil {
		return "", errors.Wrap(err, "Error getting health check probes for load balancer")
	}

	if _, err := conn.computeService.TargetPools.Get(project, region, name).Do(); err != nil {
		return "", errors.Wrap(err, "Error getting target pools for load balancer")
	}

	if _, err := conn.computeService.ForwardingRules.Get(project, region, name).Do(); err != nil {
		return "", errors.Wrap(err, "Error getting forwarding rules for load balancer")
	}

	log.Infof("Successfully found load balancer for cluster %s", conn.Cluster.Name)
	return address.Address, nil
}

func (conn *cloudConnector) createLoadBalancer(leaderMachine string) (string, error) {
	log.Infof("Creating load balancer cluster %s", conn.Cluster.Name)

	project := conn.Cluster.Spec.Config.Cloud.Project
	region := conn.Cluster.Spec.Config.Cloud.Region
	zone := conn.Cluster.Spec.Config.Cloud.Zone
	clusterName := conn.Cluster.Name
	name := conn.namer.loadBalancerName()

	addressOp, err := conn.computeService.Addresses.Insert(project, region, &compute.Address{
		Name: name,
	}).Do()

	if err == nil {
		if err := conn.waitForRegionOperation(addressOp.Name); err != nil {
			return "", errors.Wrap(err, "Timed out waiting for load balancer ip to get ready")
		}
	} else if !errAlreadyExists(err) {
		return "", errors.Wrap(err, "Error creating ip address for load balancer")
	}

	address, err := conn.computeService.Addresses.Get(project, region, name).Do()
	if err != nil {
		return "", errors.Wrap(err, "Error getting load balancer ip")
	}

	_, err = conn.computeService.HttpHealthChecks.Insert(project, &compute.HttpHealthCheck{
		Name: name,
		Port: 6443,
	}).Do()

	if err != nil && !errAlreadyExists(err) {
		return "", errors.Wrap(err, "Error creating healthcheck probes for load balancer")
	}

	targetPools, err := conn.computeService.TargetPools.Insert(project, region, &compute.TargetPool{
		Name:         name,
		HealthChecks: []string{fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/httpHealthChecks/%s", project, name)},
		Instances:    []string{fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, leaderMachine)},
	}).Do()

	if err == nil {
		if err := conn.waitForRegionOperation(targetPools.Name); err != nil {
			return "", errors.Wrap(err, "Timed out waiting for backend pools to be ready")
		}
	} else if !errAlreadyExists(err) {
		return "", errors.Wrap(err, "Error creating load balancer backend pools")
	}

	forwardingRule, err := conn.computeService.ForwardingRules.Insert(project, region, &compute.ForwardingRule{
		LoadBalancingScheme: "EXTERNAL",
		IPAddress:           address.Address,
		Name:                name,
		IPProtocol:          "TCP",
		PortRange:           "6443",
		Target:              fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools/%s", project, region, name),
	}).Do()

	if err == nil {
		if err := conn.waitForRegionOperation(forwardingRule.Name); err != nil {
			return "", errors.Wrap(err, "Timed out waiting for load balancer forwarding rules to be ready")
		}
	} else if !errAlreadyExists(err) {
		return "", errors.Wrap(err, "Error creating load balancer forwarding rules")
	}

	log.Infof("Successfully created load balancer: %s-apiserver with ip %q", clusterName, address.Address)
	return address.Address, nil
}

func (conn *cloudConnector) deleteLoadBalancer() error {
	log.Infof("Deleting load balancer")

	project := conn.Cluster.Spec.Config.Cloud.Project
	region := conn.Cluster.Spec.Config.Cloud.Region
	clusterName := conn.Cluster.Name
	name := fmt.Sprintf("%s-apiserver", clusterName)

	if _, err := conn.computeService.Addresses.Get(project, region, name).Do(); err == nil {
		if _, err := conn.computeService.Addresses.Delete(project, region, name).Do(); err != nil {
			return errors.Wrap(err, "failed to delete ip of load balancre")
		}
	} else if !errDeleted(err) {
		return errors.Wrap(err, "Error getting load balancer ip address")
	}
	log.Info("deleted lb ip address")

	if _, err := conn.computeService.ForwardingRules.Get(project, region, name).Do(); err == nil {
		deleteOp, err := conn.computeService.ForwardingRules.Delete(project, region, name).Do()
		if err != nil {
			return errors.Wrap(err, "failed to delete forwarding rules for load balancer")
		}
		if err := conn.waitForRegionOperation(deleteOp.Name); err != nil {
			return errors.Wrap(err, "timed out deleting lb forwarding rule")
		}
	} else if !errDeleted(err) {
		return errors.Wrap(err, "Error getting forwarding rules for load balancer")
	}
	log.Info("deleted lb forwarding rule")

	if _, err := conn.computeService.TargetPools.Get(project, region, name).Do(); err == nil {
		deleteOp, err := conn.computeService.TargetPools.Delete(project, region, name).Do()
		if err != nil {
			return errors.Wrap(err, "failed to delete load balancer target pool")
		}
		if err := conn.waitForRegionOperation(deleteOp.Name); err != nil {
			return errors.Wrap(err, "timed out deleting lb targte pool")
		}
	} else if !errDeleted(err) {
		return errors.Wrap(err, "Error getting target pools for load balancer")
	}
	log.Info("deleted lb target rule")

	if _, err := conn.computeService.HttpHealthChecks.Get(project, name).Do(); err == nil {
		if _, err := conn.computeService.HttpHealthChecks.Delete(project, name).Do(); err != nil {
			return errors.Wrap(err, "failed to delete load balancer health check probes")
		}
	} else if !errDeleted(err) {
		return errors.Wrap(err, "Error getting health check probes for load balancer")
	}
	log.Info("deleted lb health check probe")

	log.Infoln("successfully deleted load balancer")
	return nil
}

func (conn *cloudConnector) deleteInstance(name string) error {
	log.Info("Deleting instance...")
	r, err := conn.computeService.Instances.Delete(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, name).Do()
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = conn.waitForZoneOperation(r.Name)
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) waitForGlobalOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.GlobalOperations.Get(conn.Cluster.Spec.Config.Cloud.Project, operation).Do()
		if err != nil && errDeleted(err) {
			return false, nil
		}
		log.Infof("Attempt %v: Operation %v is %v ...", attempt, operation, r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) waitForRegionOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.RegionOperations.Get(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Region, operation).Do()
		if err != nil && !errDeleted(err) {
			log.Info(err)
			return false, nil
		} else if err != nil && errDeleted(err) {
			return true, nil
		}
		log.Infof("Attempt %v: Operation %v is in state %v ...", attempt, operation, r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) waitForZoneOperation(operation string) error {
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.computeService.ZoneOperations.Get(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, operation).Do()
		if err != nil && !errDeleted(err) {
			return false, nil
		} else if err != nil && errDeleted(err) {
			return true, nil
		}
		log.Infof("Attempt %v: Operation %v is %v ...", attempt, operation, r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func errDeleted(err error) bool {
	return strings.Contains(err.Error(), "notFound")
}

func (conn *cloudConnector) importPublicKey() error {
	pubKey := string(conn.Certs.SSHKey.PublicKey)
	r1, err := conn.computeService.Projects.SetCommonInstanceMetadata(conn.Cluster.Spec.Config.Cloud.Project, &compute.Metadata{
		Items: []*compute.MetadataItems{
			{
				Key:   conn.Cluster.Spec.Config.Cloud.SSHKeyName,
				Value: &pubKey,
			},
		},
	}).Do()
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = conn.waitForGlobalOperation(r1.Name)
	if err != nil {
		return errors.Wrap(err, "")
	}
	log.Debug("Imported SSH key")
	log.Info("SSH key imported")
	return nil
}

func (conn *cloudConnector) getNetworks() (bool, error) {
	log.Infof("Retrieving network %v for project %v", defaultNetwork, conn.Cluster.Spec.Config.Cloud.Project)
	r1, err := conn.computeService.Networks.Get(conn.Cluster.Spec.Config.Cloud.Project, defaultNetwork).Do()
	log.Debug("Retrieve network result", r1, err)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) ensureNetworks() error {
	log.Infof("Retrieving network %v for project %v", defaultNetwork, conn.Cluster.Spec.Config.Cloud.Project)
	r2, err := conn.computeService.Networks.Insert(conn.Cluster.Spec.Config.Cloud.Project, &compute.Network{
		IPv4Range: "10.240.0.0/16",
		Name:      defaultNetwork,
	}).Do()
	log.Debug("Created new network", r2, err)
	if err != nil {
		return errors.Wrap(err, "")
	}
	log.Infof("New network %v is created", defaultNetwork)

	return nil
}

func (conn *cloudConnector) getFirewallRules() (bool, error) {
	ruleClusterInternal := conn.Cluster.Name + firewallRuleInternalSuffix
	log.Infof("Retrieving firewall rule %v", ruleClusterInternal)
	if r1, err := conn.computeService.Firewalls.Get(conn.Cluster.Spec.Config.Cloud.Project, ruleClusterInternal).Do(); err != nil {
		log.Debug("Retrieved firewall rule", r1, err)
		return false, err
	}

	ruleSSH := conn.Cluster.Name + "-allow-ssh"
	if r2, err := conn.computeService.Firewalls.Get(conn.Cluster.Spec.Config.Cloud.Project, ruleSSH).Do(); err != nil {
		log.Debug("Retrieved firewall rule", r2, err)
		return false, err
	}

	ruleApiPublic := conn.Cluster.Name + firewallRuleApiSuffix
	if r5, err := conn.computeService.Firewalls.Get(conn.Cluster.Spec.Config.Cloud.Project, ruleApiPublic).Do(); err != nil {
		log.Debug("Retrieved firewall rule", r5, err)
		return false, err
	}

	return true, nil
}

func (conn *cloudConnector) ensureFirewallRules() error {
	network := fmt.Sprintf("projects/%v/global/networks/%v", conn.Cluster.Spec.Config.Cloud.Project, defaultNetwork)
	var err error
	cluster := conn.Cluster
	ruleInternal := defaultNetwork + "-allow-internal"
	if r1, err := conn.computeService.Firewalls.Get(conn.Cluster.Spec.Config.Cloud.Project, ruleInternal).Do(); err != nil {
		log.Debug("Retrieved firewall rule", r1, err)

		r2, err := conn.computeService.Firewalls.Insert(conn.Cluster.Spec.Config.Cloud.Project, &compute.Firewall{
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
		log.Debug("Created firewall rule", r2, err)
		if err != nil {
			return errors.Wrap(err, "")
		}
		log.Infof("Firewall rule %v created", ruleInternal)
	}

	ruleClusterInternal := cluster.Name + firewallRuleInternalSuffix

	if r1, err := conn.computeService.Firewalls.Get(cluster.Spec.Config.Cloud.Project, ruleClusterInternal).Do(); err != nil {
		log.Debug("Retrieved firewall rule", r1, err)

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

		log.Debug("Created firewall rule", r2, err)
		if err != nil {
			return errors.Wrap(err, "")
		}
		log.Infof("Firewall rule %v created", ruleClusterInternal)
	}
	if cluster.Spec.ClusterAPI.ObjectMeta.Annotations == nil {
		cluster.Spec.ClusterAPI.ObjectMeta.Annotations = make(map[string]string)
	}
	cluster.Spec.ClusterAPI.ObjectMeta.Annotations[firewallRuleAnnotationPrefix+ruleClusterInternal] = "true"
	if conn.Cluster, err = store.StoreProvider.Clusters().Update(cluster); err != nil {
		return err
	}

	ruleSSH := cluster.Name + "-allow-ssh"
	if r3, err := conn.computeService.Firewalls.Get(conn.Cluster.Spec.Config.Cloud.Project, ruleSSH).Do(); err != nil {
		log.Debug("Retrieved firewall rule", r3, err)

		r4, err := conn.computeService.Firewalls.Insert(conn.Cluster.Spec.Config.Cloud.Project, &compute.Firewall{
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
		log.Debug("Created firewall rule", r4, err)
		if err != nil {
			return errors.Wrap(err, "")
		}
		log.Infof("Firewall rule %v created", ruleSSH)
	}

	ruleApiPublic := cluster.Name + firewallRuleApiSuffix

	if r7, err := conn.computeService.Firewalls.Get(conn.Cluster.Spec.Config.Cloud.Project, ruleApiPublic).Do(); err != nil {
		log.Debug("Retrieved firewall rule", r7, err)
		r8, err := conn.computeService.Firewalls.Insert(conn.Cluster.Spec.Config.Cloud.Project, &compute.Firewall{
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

		log.Debug("Created firewall rule", r8, err)

		if err != nil {
			return errors.Wrap(err, "")
		}
		log.Infof("Firewall rule %v created", ruleSSH)
	}

	cluster.Spec.ClusterAPI.ObjectMeta.Annotations[firewallRuleAnnotationPrefix+ruleApiPublic] = "true"
	if conn.Cluster, err = store.StoreProvider.Clusters().Update(cluster); err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) getMasterPDDisk(name string) (bool, error) {

	if r, err := conn.computeService.Disks.Get(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, name).Do(); err != nil {
		log.Debug("Retrieved master persistent disk", r, err)
		return false, err
	}

	//conn.Cluster.Spec.MasterDiskId = name
	return true, nil
}

func (conn *cloudConnector) createDisk(name, diskType string, sizeGb int64) (string, error) {
	// Type:        "https://www.googleapis.com/compute/v1/projects/tigerworks-kube/zones/us-central1-b/diskTypes/pd-ssd",
	// SourceImage: "https://www.googleapis.com/compute/v1/projects/google-containers/global/images/container-vm-v20150806",

	dType := fmt.Sprintf("projects/%v/zones/%v/diskTypes/%v", conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, diskType)

	r1, err := conn.computeService.Disks.Insert(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, &compute.Disk{
		Name:   name,
		Zone:   conn.Cluster.Spec.Config.Cloud.Zone,
		Type:   dType,
		SizeGb: sizeGb,
	}).Do()

	log.Debug("Created master disk", r1, err)
	if err != nil {
		return name, errors.Wrap(err, "")
	}
	err = conn.waitForZoneOperation(r1.Name)
	if err != nil {
		return name, errors.Wrap(err, "")
	}
	log.Infof("Blank disk of type %v created before creating the master VM", dType)
	return name, nil
}

func (conn *cloudConnector) getMasterInstance(machine *clusterv1.Machine) (bool, error) {
	if r1, err := conn.computeService.Instances.Get(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, machine.Name).Do(); err != nil {
		log.Debug("Retrieved master instance", r1, err)
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) createMasterIntance(cluster *api.Cluster, script string) (string, error) {
	// MachineType:  "projects/tigerworks-kube/zones/us-central1-b/machineTypes/n1-standard-1",
	// Zone:         "projects/tigerworks-kube/zones/us-central1-b",

	machine, err := GetLeaderMachine(conn.Cluster)
	if err != nil {
		return "", errors.Wrap(err, "failed to get leader machine")
	}

	if found, _ := conn.getMasterPDDisk(conn.namer.MachineDiskName(machine)); !found {
		_, err := conn.createDisk(conn.namer.MachineDiskName(machine), "pd-standard", 30)
		if err != nil {
			return "", errors.Wrap(err, "failed to create disk for master machine")
		}
	}

	providerSpec, err := proconfig.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return "", errors.Wrap(err, "Error decoding machine provider spec from machine spec")
	}

	machineType := fmt.Sprintf("projects/%v/zones/%v/machineTypes/%v", conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, providerSpec.MachineType)
	zone := fmt.Sprintf("projects/%v/zones/%v", conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone)
	pdSrc := fmt.Sprintf("projects/%v/zones/%v/disks/%v", conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, conn.namer.MachineDiskName(machine))
	srcImage := fmt.Sprintf("projects/%v/global/images/%v", conn.Cluster.Spec.Config.Cloud.InstanceImageProject, conn.Cluster.Spec.Config.Cloud.InstanceImage)

	pubKey := string(conn.Certs.SSHKey.PublicKey)
	value := fmt.Sprintf("%v:%v %v", conn.namer.AdminUsername(), pubKey, conn.namer.AdminUsername())

	instance := &compute.Instance{
		Name:        machine.Name,
		Zone:        zone,
		MachineType: machineType,
		Tags: &compute.Tags{
			Items: []string{
				"https-server",
				conn.Cluster.Name + "-master",
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Network: fmt.Sprintf("projects/%v/global/networks/%v", conn.Cluster.Spec.Config.Cloud.Project, defaultNetwork),
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

	r1, err := conn.computeService.Instances.Insert(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, instance).Do()
	log.Debug("Created master instance", r1, err)

	if err != nil {
		return instance.Name, errors.Wrap(err, "")
	}
	//log.Infof("Master instance of type %v in zone %v using persistent disk %v created", machineType, zone, pdSrc)
	log.Infof("Master instance of type %v in zone %v is created", machineType, zone)

	return r1.Name, nil
}

func (conn *cloudConnector) deleteMaster(machines []*clusterv1.Machine) error {
	size := len(machines)
	errorCh := make(chan error, size)

	var wg sync.WaitGroup

	wg.Add(size)

	for _, machine := range machines {
		go func(machine *clusterv1.Machine) {
			defer wg.Done()

			log.Infof("Deleting machine %s", machine.Name)

			instance, err := conn.computeService.Instances.Delete(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, machine.Name).Do()
			if err != nil {
				errorCh <- errors.Wrapf(err, "failed to delete machine %s", machine.Name)
				return
			}

			// wait for instances to be deleted
			// after they're completely deleted, we can remove attached disks
			if err := conn.waitForZoneOperation(instance.Name); err != nil {
				errorCh <- errors.Wrapf(err, "timed out waiting for deleting machine %s", machine.Name)
				return
			}

			log.Infof("successfully deleted machine %s", machine.Name)
		}(machine)
	}
	wg.Wait()
	close(errorCh)

	var allerr error
	for err := range errorCh {
		if allerr == nil {
			allerr = errors.New("failed to delete master machines")
		}
		allerr = errors.Wrapf(allerr, "%v", err)
	}

	if allerr == nil {
		log.Info("successfully deleted master machines")
		return nil
	}
	return errors.Wrap(allerr, "failed to delete master machines")
}

//delete disk
func (conn *cloudConnector) deleteDisk(nodeDiskNames []string) error {
	log.Info("Deleting disks")

	project := conn.Cluster.Spec.Config.Cloud.Project
	zone := conn.Cluster.Spec.Config.Cloud.Zone

	var wg sync.WaitGroup
	wg.Add(len(nodeDiskNames))

	errCh := make(chan error, len(nodeDiskNames))

	for _, disk := range nodeDiskNames {
		go func(name string) {
			log.Infof("deleting disk %s", name)

			defer wg.Done()

			if _, err := conn.computeService.Disks.Delete(project, zone, name).Do(); err != nil && !errDeleted(err) {
				log.Infof("failed to delete disk %s: %v", name, err)
				errCh <- err
				return
			}

			log.Infof("successfully deleted disk %s", name)
		}(disk)
	}
	wg.Wait()
	close(errCh)

	var allerr error
	for err := range errCh {
		if allerr == nil {
			allerr = errors.New("failed to delete master machines")
		}
		allerr = errors.Wrap(allerr, fmt.Sprintf("%v", err))
	}

	if allerr == nil {
		log.Info("successfully deleted disks")
		return nil
	}

	return allerr
}

//delete firewalls
// ruleClusterInternal := cluster.Name + firewallRuleInternalSuffix
// ruleSSH := cluster.Name + "-allow-ssh"
// ruleApiPublic := cluster.Name + firewallRuleApiSuffix

func (conn *cloudConnector) deleteFirewalls() error {
	time.Sleep(3 * time.Second)

	ruleClusterInternal := conn.Cluster.Name + firewallRuleInternalSuffix
	r2, err := conn.computeService.Firewalls.Delete(conn.Cluster.Spec.Config.Cloud.Project, ruleClusterInternal).Do()
	if err != nil {
		log.Infoln(err)
		//return errors.Wrap(err, "")
	} else {
		log.Infof("Firewalls %v deleted, response %v", ruleClusterInternal, r2.Status)
	}
	time.Sleep(3 * time.Second)

	ruleSSH := conn.Cluster.Name + "-allow-ssh"
	r3, err := conn.computeService.Firewalls.Delete(conn.Cluster.Spec.Config.Cloud.Project, ruleSSH).Do()
	if err != nil {
		log.Infoln(err)
		//return errors.Wrap(err, "")
	} else {
		log.Infof("Firewalls %v deleted, response %v", ruleSSH, r3.Status)
	}
	time.Sleep(3 * time.Second)

	ruleApiPublic := conn.Cluster.Name + firewallRuleApiSuffix
	r4, err := conn.computeService.Firewalls.Delete(conn.Cluster.Spec.Config.Cloud.Project, ruleApiPublic).Do()
	if err != nil {
		log.Infoln(err)
		//return errors.Wrap(err, "")
	} else {
		log.Infof("Firewalls %v deleted, response %v", ruleApiPublic, r4.Status)
	}
	time.Sleep(3 * time.Second)
	return nil
}
