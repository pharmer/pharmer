package gce

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	compute "google.golang.org/api/compute/v1"
)

func (cm *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	var err error

	cm.cluster, err = NewCluster(req)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.namer = namer{cluster: cm.cluster}
	if _, err := cloud.Store(cm.ctx).Clusters().Create(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.conn, err = NewConnector(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)

	defer func(releaseReservedIp bool) {
		if cm.cluster.Status.Phase == api.ClusterPhasePending {
			cm.cluster.Status.Phase = api.ClusterPhaseFailing
		}
		cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		cloud.Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterPhaseReady {
			cloud.Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.Delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")

	//if cm.cluster.Spec.InstanceImage, err = cm.conn.getInstanceImage(); err != nil {
	//	cm.cluster.Status.Reason = err.Error()
	//	return errors.FromErr(err).WithContext(cm.ctx).Err()
	//}

	if err = cm.importPublicKey(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// TODO: Should we add *IfMissing suffix to all these functions
	if err = cm.ensureNetworks(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.ensureFirewallRules(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Spec.MasterDiskId, err = cm.createDisk(cm.namer.MasterPDName(), cm.cluster.Spec.MasterDiskType, cm.cluster.Spec.MasterDiskSize)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.reserveIP(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if cm.ctx, err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// needed for master start-up config
	if _, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	op1, err := cm.createMasterIntance()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForZoneOperation(op1)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterInstance, err := cm.getInstance(cm.cluster.Spec.KubernetesMasterName)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Spec.Role = api.RoleKubernetesMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.Spec.MasterExternalIP)
	cloud.Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// needed for node start-up config to get master_internal_ip
	if _, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// Use zone operation to wait and block.
	if op2, err := cm.createNodeFirewallRule(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	} else {
		if err = cm.conn.waitForGlobalOperation(op2); err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	for _, ng := range req.NodeGroups {
		igm := &InstanceGroupManager{
			cm: cm,
			instance: cloud.Instance{
				Type: cloud.InstanceType{
					ContextVersion: cm.cluster.Spec.ResourceVersion,
					Sku:            ng.Sku,

					Master:       false,
					SpotInstance: false,
				},
				Stats: cloud.GroupStats{
					Count: ng.Count,
				},
			},
		}
		igm.AdjustInstanceGroup()
	}
	cloud.Logger(cm.ctx).Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := cloud.EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// wait for nodes to start
	if err := cloud.WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// -------------------------------------------------------------------------------------------------------------

	time.Sleep(time.Minute * 1)
	cloud.Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)
	for _, ng := range req.NodeGroups {
		instances, err := cm.listInstances(cm.namer.InstanceGroupName(ng.Sku))
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		for _, node := range instances {
			cloud.Store(cm.ctx).Instances(cm.cluster.Name).Create(node)
		}
	}

	cm.cluster.Status.Phase = api.ClusterPhaseReady
	return nil
}

func (cm *ClusterManager) importPublicKey() error {
	cloud.Logger(cm.ctx).Infof("Importing SSH key with fingerprint: %v", cm.cluster.Spec.SSHKey.OpensshFingerprint)
	pubKey := string(cm.cluster.Spec.SSHKey.PublicKey)
	r1, err := cm.conn.computeService.Projects.SetCommonInstanceMetadata(cm.cluster.Spec.Project, &compute.Metadata{
		Items: []*compute.MetadataItems{
			{
				Key:   cm.cluster.Spec.SSHKeyExternalID,
				Value: &pubKey,
			},
		},
	}).Do()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cm.conn.waitForGlobalOperation(r1.Name)
	if err != nil {
		errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Debug("Imported SSH key")
	cloud.Logger(cm.ctx).Info("SSH key imported")
	return nil
}

func (cm *ClusterManager) ensureNetworks() error {
	cloud.Logger(cm.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, cm.cluster.Spec.Project)
	if r1, err := cm.conn.computeService.Networks.Get(cm.cluster.Spec.Project, defaultNetwork).Do(); err != nil {
		cloud.Logger(cm.ctx).Debug("Retrieve network result", r1, err)
		r2, err := cm.conn.computeService.Networks.Insert(cm.cluster.Spec.Project, &compute.Network{
			IPv4Range: "10.240.0.0/16",
			Name:      defaultNetwork,
		}).Do()
		cloud.Logger(cm.ctx).Debug("Created new network", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("New network %v is created", defaultNetwork)
	}
	return nil
}

func (cm *ClusterManager) ensureFirewallRules() error {
	network := fmt.Sprintf("projects/%v/global/networks/%v", cm.cluster.Spec.Project, defaultNetwork)
	ruleInternal := defaultNetwork + "-allow-internal"
	cloud.Logger(cm.ctx).Infof("Retrieving firewall rule %v", ruleInternal)
	if r1, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Project, ruleInternal).Do(); err != nil {
		cloud.Logger(cm.ctx).Debug("Retrieved firewall rule", r1, err)

		r2, err := cm.conn.computeService.Firewalls.Insert(cm.cluster.Spec.Project, &compute.Firewall{
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
		cloud.Logger(cm.ctx).Debug("Created firewall rule", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Firewall rule %v created", ruleInternal)
	}

	ruleSSH := defaultNetwork + "-allow-ssh"
	if r3, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Project, ruleSSH).Do(); err != nil {
		cloud.Logger(cm.ctx).Debug("Retrieved firewall rule", r3, err)

		r4, err := cm.conn.computeService.Firewalls.Insert(cm.cluster.Spec.Project, &compute.Firewall{
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
		cloud.Logger(cm.ctx).Debug("Created firewall rule", r4, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Firewall rule %v created", ruleSSH)
	}

	ruleHTTPS := cm.cluster.Spec.KubernetesMasterName + "-https"
	if r5, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Project, ruleHTTPS).Do(); err != nil {
		cloud.Logger(cm.ctx).Debug("Retrieved firewall rule", r5, err)

		r6, err := cm.conn.computeService.Firewalls.Insert(cm.cluster.Spec.Project, &compute.Firewall{
			Name:         ruleHTTPS,
			Network:      network,
			SourceRanges: []string{"0.0.0.0/0"},
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
					Ports:      []string{"443"},
				},
				{
					IPProtocol: "tcp",
					Ports:      []string{"6443"},
				},
			},
			TargetTags: []string{cm.cluster.Name + "-master"},
		}).Do()
		cloud.Logger(cm.ctx).Debug("Created master and configuring firewalls", r6, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Info("Master created and firewalls configured")
	}
	return nil
}

func (cm *ClusterManager) createDisk(name, diskType string, sizeGb int64) (string, error) {
	// Type:        "https://www.googleapis.com/compute/v1/projects/tigerworks-kube/zones/us-central1-b/diskTypes/pd-ssd",
	// SourceImage: "https://www.googleapis.com/compute/v1/projects/google-containers/global/images/container-vm-v20150806",

	dType := fmt.Sprintf("projects/%v/zones/%v/diskTypes/%v", cm.cluster.Spec.Project, cm.cluster.Spec.Zone, diskType)

	r1, err := cm.conn.computeService.Disks.Insert(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, &compute.Disk{
		Name:   name,
		Zone:   cm.cluster.Spec.Zone,
		Type:   dType,
		SizeGb: sizeGb,
	}).Do()
	cloud.Logger(cm.ctx).Debug("Created master disk", r1, err)
	if err != nil {
		return name, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Blank disk of type %v created before creating the master VM", dType)
	return name, nil
}

func (cm *ClusterManager) reserveIP() error {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		name := cm.namer.ReserveIPName()

		cloud.Logger(cm.ctx).Infof("Checking existence of reserved master ip %v", name)
		if r1, err := cm.conn.computeService.Addresses.Get(cm.cluster.Spec.Project, cm.cluster.Spec.Region, name).Do(); err == nil {
			if r1.Status == "IN_USE" {
				return fmt.Errorf("Found a static IP with name %v in use. Failed to reserve a new ip with the same name.", name)
			}

			cloud.Logger(cm.ctx).Debug("Found master IP was already reserved", r1, err)
			cm.cluster.Spec.MasterReservedIP = r1.Address
			cloud.Logger(cm.ctx).Infof("Newly reserved master ip %v", cm.cluster.Spec.MasterReservedIP)
			return nil
		}

		cloud.Logger(cm.ctx).Infof("Reserving master ip %v", name)
		r2, err := cm.conn.computeService.Addresses.Insert(cm.cluster.Spec.Project, cm.cluster.Spec.Region, &compute.Address{Name: name}).Do()
		cloud.Logger(cm.ctx).Debug("Reserved master IP", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		err = cm.conn.waitForRegionOperation(r2.Name)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Master ip %v reserved", name)
		if r3, err := cm.conn.computeService.Addresses.Get(cm.cluster.Spec.Project, cm.cluster.Spec.Region, name).Do(); err == nil {
			cloud.Logger(cm.ctx).Debug("Retrieved newly reserved master IP", r3, err)

			cm.cluster.Spec.MasterReservedIP = r3.Address
			cloud.Logger(cm.ctx).Infof("Newly reserved master ip %v", cm.cluster.Spec.MasterReservedIP)
		}
	}

	return nil
}

func (cm *ClusterManager) createMasterIntance() (string, error) {
	// MachineType:  "projects/tigerworks-kube/zones/us-central1-b/machineTypes/n1-standard-1",
	// Zone:         "projects/tigerworks-kube/zones/us-central1-b",

	// startupScript := cm.RenderStartupScript(cm.cluster, cm.cluster.Spec.MasterSKU, api.RoleKubernetesMaster)
	startupScript, err := cloud.RenderStartupScript(cm.ctx, cm.cluster, api.RoleKubernetesMaster)
	if err != nil {
		return "", err
	}

	machineType := fmt.Sprintf("projects/%v/zones/%v/machineTypes/%v", cm.cluster.Spec.Project, cm.cluster.Spec.Zone, cm.cluster.Spec.MasterSKU)
	zone := fmt.Sprintf("projects/%v/zones/%v", cm.cluster.Spec.Project, cm.cluster.Spec.Zone)
	pdSrc := fmt.Sprintf("projects/%v/zones/%v/disks/%v", cm.cluster.Spec.Project, cm.cluster.Spec.Zone, cm.namer.MasterPDName())
	srcImage := fmt.Sprintf("projects/%v/global/images/%v", cm.cluster.Spec.InstanceImageProject, cm.cluster.Spec.InstanceImage)

	instance := &compute.Instance{
		Name:        cm.cluster.Spec.KubernetesMasterName,
		Zone:        zone,
		MachineType: machineType,
		// --image-project="${MASTER_IMAGE_PROJECT}"
		// --image "${MASTER_IMAGE}"
		Tags: &compute.Tags{
			Items: []string{cm.cluster.Name + "-master"},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Network: fmt.Sprintf("projects/%v/global/networks/%v", cm.cluster.Spec.Project, defaultNetwork),
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
					Value: &startupScript,
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
	if cm.cluster.Spec.MasterReservedIP == "" {
		instance.NetworkInterfaces[0].AccessConfigs = []*compute.AccessConfig{
			{
				Name: "Master External IP",
				Type: "ONE_TO_ONE_NAT",
			},
		}
	} else {
		instance.NetworkInterfaces[0].AccessConfigs = []*compute.AccessConfig{
			{
				NatIP: cm.cluster.Spec.MasterReservedIP,
			},
		}
	}
	r1, err := cm.conn.computeService.Instances.Insert(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, instance).Do()
	cloud.Logger(cm.ctx).Debug("Created master instance", r1, err)
	if err != nil {
		return r1.Name, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Master instance of type %v in zone %v using persistent disk %v created", machineType, zone, pdSrc)
	return r1.Name, nil
}

// Instance
func (cm *ClusterManager) getInstance(instance string) (*api.Instance, error) {
	cloud.Logger(cm.ctx).Infof("Retrieving instance %v in zone %v", instance, cm.cluster.Spec.Zone)
	r1, err := cm.conn.computeService.Instances.Get(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, instance).Do()
	cloud.Logger(cm.ctx).Debug("Retrieved instance", r1, err)
	if err != nil {
		return nil, err
	}
	return cm.newKubeInstance(r1)
}

func (cm *ClusterManager) listInstances(instanceGroup string) ([]*api.Instance, error) {
	cloud.Logger(cm.ctx).Infof("Retrieving instances in node group %v", instanceGroup)
	instances := make([]*api.Instance, 0)
	r1, err := cm.conn.computeService.InstanceGroups.ListInstances(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, instanceGroup, &compute.InstanceGroupsListInstancesRequest{
		InstanceState: "ALL",
	}).Do()
	cloud.Logger(cm.ctx).Debug("Retrieved instance", r1, err)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	for _, item := range r1.Items {
		name := item.Instance[strings.LastIndex(item.Instance, "/")+1:]
		r2, err := cm.conn.computeService.Instances.Get(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, name).Do()
		if err != nil {
			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		instance, err := cm.newKubeInstance(r2)
		if err != nil {
			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		instance.Spec.Role = api.RoleKubernetesPool
		instances = append(instances, instance)
	}
	return instances, nil
}

func (cm *ClusterManager) newKubeInstance(r1 *compute.Instance) (*api.Instance, error) {
	for _, accessConfig := range r1.NetworkInterfaces[0].AccessConfigs {
		if accessConfig.Type == "ONE_TO_ONE_NAT" {
			i := api.Instance{
				ObjectMeta: api.ObjectMeta{
					UID:  phid.NewKubeInstance(),
					Name: r1.Name,
				},
				Spec: api.InstanceSpec{
					SKU: r1.MachineType[strings.LastIndex(r1.MachineType, "/")+1:],
				},
				Status: api.InstanceStatus{
					ExternalID:    strconv.FormatUint(r1.Id, 10),
					ExternalPhase: r1.Status,
					PublicIP:      accessConfig.NatIP,
					PrivateIP:     r1.NetworkInterfaces[0].NetworkIP,
				},
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
			if r1.Status == "TERMINATED" {
				i.Status.Phase = api.InstancePhaseDeleted
			} else {
				i.Status.Phase = api.InstancePhaseReady
			}
			return &i, nil
		}
	}
	return nil, errors.New("Failed to convert gcloud instance to KubeInstance.").WithContext(cm.ctx).Err() //stackerr.New("Failed to convert gcloud instance to KubeInstance.")
}

func (cm *ClusterManager) createNodeFirewallRule() (string, error) {
	name := cm.cluster.Name + "-node-all"
	network := fmt.Sprintf("projects/%v/global/networks/%v", cm.cluster.Spec.Project, defaultNetwork)

	r1, err := cm.conn.computeService.Firewalls.Insert(cm.cluster.Spec.Project, &compute.Firewall{
		Name:         name,
		Network:      network,
		SourceRanges: []string{cm.cluster.Spec.ClusterIPRange},
		TargetTags:   []string{cm.cluster.Name + "-node"},
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
	cloud.Logger(cm.ctx).Debug("Created firewall rule", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Node firewall rule %v created", name)
	return r1.Name, nil
}

func (cm *ClusterManager) createNodeInstanceTemplate(sku string) (string, error) {
	templateName := cm.namer.InstanceTemplateName(sku)

	cloud.Logger(cm.ctx).Infof("Retrieving node template %v", templateName)
	if r1, err := cm.conn.computeService.InstanceTemplates.Get(cm.cluster.Spec.Project, templateName).Do(); err == nil {
		cloud.Logger(cm.ctx).Debug("Retrieved node template", r1, err)

		cloud.Logger(cm.ctx).Infof("Deleting node template %v", templateName)
		if r2, err := cm.conn.computeService.InstanceTemplates.Delete(cm.cluster.Spec.Project, templateName).Do(); err != nil {
			cloud.Logger(cm.ctx).Debug("Delete node template called", r2, err)
			cloud.Logger(cm.ctx).Infoln("Failed to delete existing instance template")
			os.Exit(1)
		}
	}
	//  if cluster.Spec.ctx.Preemptiblenode == "true" {
	//	  preemptible_nodes = "--preemptible --maintenance-policy TERMINATE"
	//  }

	startupScript, err := cloud.RenderStartupScript(cm.ctx, cm.cluster, api.RoleKubernetesPool)
	if err != nil {
		return "", err
	}

	image := fmt.Sprintf("projects/%v/global/images/%v", cm.cluster.Spec.Project, cm.cluster.Spec.InstanceImage)
	network := fmt.Sprintf("projects/%v/global/networks/%v", cm.cluster.Spec.Project, defaultNetwork)

	cloud.Logger(cm.ctx).Infof("Create instance template %v", templateName)
	tpl := &compute.InstanceTemplate{
		Name: templateName,
		Properties: &compute.InstanceProperties{
			MachineType: sku,
			Scheduling: &compute.Scheduling{
				AutomaticRestart:  false,
				OnHostMaintenance: "TERMINATE",
			},
			Disks: []*compute.AttachedDisk{
				{
					AutoDelete: true,
					Boot:       true,
					InitializeParams: &compute.AttachedDiskInitializeParams{
						DiskType:    cm.cluster.Spec.NodeDiskType,
						DiskSizeGb:  cm.cluster.Spec.NodeDiskSize,
						SourceImage: image,
					},
				},
			},
			Tags: &compute.Tags{
				Items: []string{cm.cluster.Name + "-node"},
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
						Value: &startupScript,
					},
				},
			},
		},
	}
	if cm.cluster.Spec.EnableNodePublicIP {
		tpl.Properties.NetworkInterfaces[0].AccessConfigs = []*compute.AccessConfig{
			{
				Name: "Node External IP",
				Type: "ONE_TO_ONE_NAT",
			},
		}
	}
	r1, err := cm.conn.computeService.InstanceTemplates.Insert(cm.cluster.Spec.Project, tpl).Do()
	cloud.Logger(cm.ctx).Debug("Create instance template called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Node instance template %v created for sku %v", templateName, sku)
	return r1.Name, nil
}

func (cm *ClusterManager) createInstanceGroup(sku string, count int64) (string, error) {
	name := cm.namer.InstanceGroupName(sku)
	template := fmt.Sprintf("projects/%v/global/instanceTemplates/%v", cm.cluster.Spec.Project, cm.namer.InstanceTemplateName(sku))

	cloud.Logger(cm.ctx).Infof("Creating instance group %v from template %v", name, template)
	r1, err := cm.conn.computeService.InstanceGroupManagers.Insert(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, &compute.InstanceGroupManager{
		Name:             name,
		BaseInstanceName: cm.cluster.Name + "-node-" + sku,
		TargetSize:       count,
		InstanceTemplate: template,
	}).Do()
	cloud.Logger(cm.ctx).Debug("Create instance group called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Instance group %v created with %v nodes of %v sku", name, count, sku)
	return r1.Name, nil
}

// Not used since Kube 1.3
func (cm *ClusterManager) createAutoscaler(sku string, count int64) (string, error) {
	name := cm.namer.InstanceGroupName(sku)
	target := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%v/zones/%v/instanceGroupManagers/%v", cm.cluster.Spec.Project, cm.cluster.Spec.Zone, name)

	cloud.Logger(cm.ctx).Infof("Creating auto scaler %v for instance group %v", name, target)
	r1, err := cm.conn.computeService.Autoscalers.Insert(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, &compute.Autoscaler{
		Name:   name,
		Target: target,
		AutoscalingPolicy: &compute.AutoscalingPolicy{
			MinNumReplicas: int64(1),
			MaxNumReplicas: count,
		},
	}).Do()
	cloud.Logger(cm.ctx).Debug("Create auto scaler called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Auto scaler %v for instance group %v created", name, target)
	return r1.Name, nil
}

func (cm *ClusterManager) GetInstance(md *api.InstanceStatus) (*api.Instance, error) {
	r2, err := cm.conn.computeService.Instances.Get(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, md.Name).Do()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	i, err := cm.newKubeInstance(r2)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return i, nil
}
