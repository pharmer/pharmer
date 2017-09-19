package gce

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	"github.com/tamalsaha/go-oneliners"
	compute "google.golang.org/api/compute/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) error {
	var err error
	if dryRun {
		action, err := cm.apply(in, api.DryRun)
		fmt.Println(err)
		jm, err := json.Marshal(action)
		fmt.Println(string(jm), err)
	} else {
		_, err = cm.apply(in, api.StdRun)
	}
	return err
}

func (cm *ClusterManager) apply(in *api.Cluster, rt api.RunType) (acts []api.Action, err error) {
	var (
		clusterDelete = false
	)
	if in.DeletionTimestamp != nil && in.Status.Phase != api.ClusterDeleted {
		clusterDelete = true
	}

	acts = make([]api.Action, 0)
	cm.cluster = in
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return
	}

	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return
	}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster); err != nil {
		return
	}
	cm.namer = namer{cluster: cm.cluster}

	if rt != api.DryRun {
		if err = cm.importPublicKey(); err != nil {
			cm.cluster.Status.Reason = err.Error()
			err = errors.FromErr(err).WithContext(cm.ctx).Err()
			return
		}
	}

	// TODO: Should we add *IfMissing suffix to all these functions
	if found, _ := cm.getNetworks(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Default Network",
			Message:  "Not found, will add default network with ipv4 range 10.240.0.0/16",
		})
		if rt != api.DryRun {
			if err = cm.ensureNetworks(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Default Network",
			Message:  "Found default network with ipv4 range 10.240.0.0/16",
		})
	}

	if found, _ := cm.getFirewallRules(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Default Firewall rule",
			Message:  "default-allow-internal, default-allow-ssh, https rules will be created",
		})
		if rt != api.DryRun {
			if err = cm.ensureFirewallRules(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Default Firewall rule",
			Message:  "default-allow-internal, default-allow-ssh, https rules found",
		})
	}

	nodeGroups, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	for _, node := range nodeGroups {
		if node.IsMaster() {
			if found, _ := cm.getMasterPDDisk(cm.namer.MasterPDName()); !found {
				acts = append(acts, api.Action{
					Action:   api.ActionAdd,
					Resource: "Master persistant disk",
					Message:  fmt.Sprintf("Not found, will be added with disk type %v, size %v and name %v", node.Spec.Template.Spec.DiskType, node.Spec.Template.Spec.DiskSize, cm.namer.MasterPDName()),
				})
				if rt == api.DryRun {
					break
				}
				cm.cluster.Spec.MasterDiskId, err = cm.createDisk(cm.namer.MasterPDName(), node.Spec.Template.Spec.DiskType, node.Spec.Template.Spec.DiskSize)
				if err != nil {
					cm.cluster.Status.Reason = err.Error()
					err = errors.FromErr(err).WithContext(cm.ctx).Err()
					return
				}
			} else {
				acts = append(acts, api.Action{
					Action:   api.ActionNOP,
					Resource: "Master persistant disk",
					Message:  fmt.Sprintf("Found master persistant disk with disk type %v, size %v and name %v", node.Spec.Template.Spec.DiskType, node.Spec.Template.Spec.DiskSize, cm.namer.MasterPDName()),
				})

			}
			cm.cluster.Spec.MasterSKU = node.Spec.Template.Spec.SKU
			break
		}
	}

	if found, _ := cm.getReserveIP(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Reserve IP",
			Message:  fmt.Sprintf("Not found, MasterReservedIP = ", cm.cluster.Spec.MasterReservedIP),
		})
		if rt != api.DryRun {
			if err = cm.reserveIP(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Reserve IP",
			Message:  fmt.Sprintf("Found, MasterReservedIP = ", cm.cluster.Spec.MasterReservedIP),
		})
	}

	// needed for master start-up config
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	//// check for instance count
	//ig.Spec.SKU = "n1-standard-1"
	//if totalNodes > 5 {
	//	ig.Spec.SKU = "n1-standard-2"
	//}
	//if totalNodes > 10 {
	//	ig.Spec.SKU = "n1-standard-4"
	//}
	//if totalNodes > 100 {
	//	ig.Spec.SKU = "n1-standard-8"
	//}
	//if totalNodes > 250 {
	//	ig.Spec.SKU = "n1-standard-16"
	//}
	//if totalNodes > 500 {
	//	ig.Spec.SKU = "n1-standard-32"
	//}

	if found, _ := cm.getMasterInstance(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Master instance with name %v will be created", cm.cluster.Spec.KubernetesMasterName),
		})

		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "A Record",
			Message:  fmt.Sprintf("Will create cluster apps A record %v, External domain %v and internal domain %v", Extra(cm.ctx).Domain(cm.cluster.Name), Extra(cm.ctx).ExternalDomain(cm.cluster.Name), Extra(cm.ctx).InternalDomain(cm.cluster.Name)),
		})
		oneliners.FILE()
		if rt != api.DryRun {
			oneliners.FILE()
			if op1, err := cm.createMasterIntance(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return acts, err
			} else {
				if err = cm.conn.waitForZoneOperation(op1); err != nil {
					cm.cluster.Status.Reason = err.Error()
					err = errors.FromErr(err).WithContext(cm.ctx).Err()
					return acts, err
				}
			}

			masterInstance, err := cm.getInstance(cm.cluster.Spec.KubernetesMasterName)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return acts, err
			}
			masterInstance.Spec.Role = api.RoleMaster
			cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
			cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP

			Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

			err = EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return acts, err
			}

			Logger(cm.ctx).Info("Waiting for cluster initialization")

			// Wait for master A record to propagate
			if err = EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return acts, err
			}
			// wait for nodes to start
			if err = WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return acts, err
			}
			// -------------------------------------------------------------------------------------------------------------
			master, _ := Store(cm.ctx).NodeGroups(cm.cluster.Name).Get("master")
			master.Status.Nodes = int32(1)
			Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(master)
			time.Sleep(time.Minute * 1)
			//Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(mas)

		}

	} else {
		oneliners.FILE()
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Found master instance with name %v", cm.cluster.Spec.KubernetesMasterName),
		})

		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "A Record",
			Message:  fmt.Sprintf("Found cluster apps A record %v, External domain %v and internal domain %v", Extra(cm.ctx).Domain(cm.cluster.Name), Extra(cm.ctx).ExternalDomain(cm.cluster.Name), Extra(cm.ctx).InternalDomain(cm.cluster.Name)),
		})
		masterInstance, err := cm.getInstance(cm.cluster.Spec.KubernetesMasterName)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			err = errors.FromErr(err).WithContext(cm.ctx).Err()
			return acts, err
		}
		masterInstance.Spec.Role = api.RoleMaster
		cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
		cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
		if clusterDelete {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Master Instance",
				Message:  fmt.Sprintf("Will delete master instance with name %v", cm.cluster.Spec.KubernetesMasterName),
			})
			if rt != api.DryRun {
				cm.cluster.Status.Phase = api.ClusterDeleting
				Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
				if err := cm.deleteMaster(); err != nil {
					cm.cluster.Status.Reason = err.Error()
				}
			}

			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Master persistant disk",
				Message:  fmt.Sprintf("Will delete master persistant with name %v", cm.namer.MasterPDName()),
			})

			if rt != api.DryRun {
				if err := cm.deleteDisk(); err != nil {
					cm.cluster.Status.Reason = err.Error()
				}
			}

			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Route",
				Message:  fmt.Sprintf("Route will be delete"),
			})
			if rt != api.DryRun {
				if err := cm.deleteRoutes(); err != nil {
					cm.cluster.Status.Reason = err.Error()
				}
			}

			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Route",
				Message:  fmt.Sprintf("Cluster apps A record %v, External domain %v and internal domain %v will be deleted", Extra(cm.ctx).Domain(cm.cluster.Name), Extra(cm.ctx).ExternalDomain(cm.cluster.Name), Extra(cm.ctx).InternalDomain(cm.cluster.Name)),
			})
			if rt != api.DryRun {
				if err := DeleteARecords(cm.ctx, cm.cluster); err != nil {
					cm.cluster.Status.Reason = err.Error()
				}
				masterInstance.Status.Phase = api.NodeDeleted
				Store(cm.ctx).Instances(cm.cluster.Name).Update(masterInstance)
			}
		}
	}
	oneliners.FILE()

	// needed for node start-up config to get master_internal_ip
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	if found, _ := cm.getNodeFirewallRule(); !found {
		oneliners.FILE()
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Node Firewall Rule",
			Message:  fmt.Sprintf("%v node firewall rule will be created", cm.cluster.Name+"-node-all"),
		})
		if rt != api.DryRun {
			// Use zone operation to wait and block.
			if op2, err := cm.createNodeFirewallRule(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return acts, err
			} else {
				if err = cm.conn.waitForGlobalOperation(op2); err != nil {
					cm.cluster.Status.Reason = err.Error()
					err = errors.FromErr(err).WithContext(cm.ctx).Err()
					return acts, err
				}
			}
		}
	} else if clusterDelete {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Node Firewall Rule",
			Message:  fmt.Sprintf("%v node firewall rule will be deleted", cm.cluster.Name+"-node-all"),
		})
		if rt != api.DryRun {
			if err := cm.deleteFirewalls(); err != nil {
				cm.cluster.Status.Reason = err.Error()
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Node Firewall Rule",
			Message:  fmt.Sprintf("Found %v node firewall rule", cm.cluster.Name+"-node-all"),
		})
	}

	for _, node := range nodeGroups {
		if node.IsMaster() {
			continue
		}
		igm := &NodeGroupManager{
			cm: cm,
			instance: Instance{
				Type: InstanceType{
					Sku:          node.Spec.Template.Spec.SKU,
					Master:       false,
					SpotInstance: false,
				},
				Stats: GroupStats{
					Count: node.Spec.Nodes,
				},
			},
		}
		if clusterDelete || node.DeletionTimestamp != nil {
			instanceGroupName := cm.namer.NodeGroupName(node.Spec.Template.Spec.SKU)
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Node Group",
				Message:  fmt.Sprintf("Node group %v  will be deleted", instanceGroupName),
			})
			if rt != api.DryRun {
				if err = cm.deleteNodeGroup(instanceGroupName); err != nil {
					fmt.Println(err)
				}
			}
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Autoscaler",
				Message:  fmt.Sprintf("Autoscaler %v  will be deleted", instanceGroupName),
			})
			if rt != api.DryRun {
				if err = cm.deleteAutoscaler(instanceGroupName); err != nil {
					fmt.Println(err)
				}
			}
			templateName := cm.namer.InstanceTemplateName(node.Spec.Template.Spec.SKU)
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Instance Template",
				Message:  fmt.Sprintf("Instance template %v  will be deleted", templateName),
			})
			if rt != api.DryRun {
				if err = cm.deleteInstanceTemplate(templateName); err != nil {
					fmt.Println(err)
				}
				Store(cm.ctx).NodeGroups(cm.cluster.Name).Delete(node.Name)
			}
		} else {
			oneliners.FILE()
			act, _ := igm.AdjustNodeGroup(rt)
			acts = append(acts, act...)
			if rt != api.DryRun {
				node.Status.Nodes = (int32)(node.Spec.Nodes)
				Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(node)
			}
		}
	}

	oneliners.FILE()
	if rt != api.DryRun {
		time.Sleep(1 * time.Minute)

		for _, ng := range nodeGroups {
			groupName := cm.namer.NodeGroupName(ng.Spec.Template.Spec.SKU)
			providerInstances, _ := cm.listInstances(groupName)

			runningInstance := make(map[string]*api.Node)
			for _, node := range providerInstances {
				runningInstance[node.Name] = node
			}

			clusterInstance, _ := GetClusterIstance(cm.ctx, cm.cluster, groupName)
			for _, node := range clusterInstance {
				if _, found := runningInstance[node]; !found {
					DeleteClusterInstance(cm.ctx, cm.cluster, node)
				}
			}
		}

		//for _, ng := range req.NodeGroups {
		//	instances, err := cm.listInstances(cm.namer.NodeGroupName(ng.Sku))
		//	if err != nil {
		//		cm.cluster.Status.Reason = err.Error()
		//		return errors.FromErr(err).WithContext(cm.ctx).Err()
		//	}
		//	for _, node := range instances {
		//		Store(cm.ctx).Instances(cm.cluster.Name).Create(node)
		//	}
		//}

		if !clusterDelete {
			cm.cluster.Status.Phase = api.ClusterReady
		} else {
			cm.cluster.Status.Phase = api.ClusterDeleted
		}
		Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		Store(cm.ctx).Clusters().Update(cm.cluster)
	}
	Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
	return
}

func (cm *ClusterManager) importPublicKey() error {
	Logger(cm.ctx).Infof("Importing SSH key with fingerprint: %v", SSHKey(cm.ctx).OpensshFingerprint)
	pubKey := string(SSHKey(cm.ctx).PublicKey)
	r1, err := cm.conn.computeService.Projects.SetCommonInstanceMetadata(cm.cluster.Spec.Cloud.Project, &compute.Metadata{
		Items: []*compute.MetadataItems{
			{
				Key:   cm.cluster.Status.SSHKeyExternalID,
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
	Logger(cm.ctx).Debug("Imported SSH key")
	Logger(cm.ctx).Info("SSH key imported")
	return nil
}

func (cm *ClusterManager) getNetworks() (bool, error) {
	Logger(cm.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, cm.cluster.Spec.Cloud.Project)
	r1, err := cm.conn.computeService.Networks.Get(cm.cluster.Spec.Cloud.Project, defaultNetwork).Do()
	Logger(cm.ctx).Debug("Retrieve network result", r1, err)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (cm *ClusterManager) ensureNetworks() error {
	Logger(cm.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, cm.cluster.Spec.Cloud.Project)
	r2, err := cm.conn.computeService.Networks.Insert(cm.cluster.Spec.Cloud.Project, &compute.Network{
		IPv4Range: "10.240.0.0/16",
		Name:      defaultNetwork,
	}).Do()
	Logger(cm.ctx).Debug("Created new network", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("New network %v is created", defaultNetwork)

	return nil
}

func (cm *ClusterManager) getFirewallRules() (bool, error) {
	ruleInternal := defaultNetwork + "-allow-internal"
	Logger(cm.ctx).Infof("Retrieving firewall rule %v", ruleInternal)
	if r1, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Cloud.Project, ruleInternal).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved firewall rule", r1, err)
		return false, err
	}

	ruleSSH := defaultNetwork + "-allow-ssh"
	if r2, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Cloud.Project, ruleSSH).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved firewall rule", r2, err)
		return false, err
	}
	ruleHTTPS := cm.cluster.Spec.KubernetesMasterName + "-https"
	if r3, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Cloud.Project, ruleHTTPS).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved firewall rule", r3, err)
		return false, err

	}
	return true, nil
}

func (cm *ClusterManager) ensureFirewallRules() error {
	network := fmt.Sprintf("projects/%v/global/networks/%v", cm.cluster.Spec.Cloud.Project, defaultNetwork)
	ruleInternal := defaultNetwork + "-allow-internal"
	Logger(cm.ctx).Infof("Retrieving firewall rule %v", ruleInternal)
	if r1, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Cloud.Project, ruleInternal).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved firewall rule", r1, err)

		r2, err := cm.conn.computeService.Firewalls.Insert(cm.cluster.Spec.Cloud.Project, &compute.Firewall{
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
		Logger(cm.ctx).Debug("Created firewall rule", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		Logger(cm.ctx).Infof("Firewall rule %v created", ruleInternal)
	}

	ruleSSH := defaultNetwork + "-allow-ssh"
	if r3, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Cloud.Project, ruleSSH).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved firewall rule", r3, err)

		r4, err := cm.conn.computeService.Firewalls.Insert(cm.cluster.Spec.Cloud.Project, &compute.Firewall{
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
		Logger(cm.ctx).Debug("Created firewall rule", r4, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		Logger(cm.ctx).Infof("Firewall rule %v created", ruleSSH)
	}

	ruleHTTPS := cm.cluster.Spec.KubernetesMasterName + "-https"
	if r5, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Cloud.Project, ruleHTTPS).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved firewall rule", r5, err)

		r6, err := cm.conn.computeService.Firewalls.Insert(cm.cluster.Spec.Cloud.Project, &compute.Firewall{
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
		Logger(cm.ctx).Debug("Created master and configuring firewalls", r6, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		Logger(cm.ctx).Info("Master created and firewalls configured")
	}
	return nil
}

func (cm *ClusterManager) getMasterPDDisk(name string) (bool, error) {
	if r, err := cm.conn.computeService.Disks.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, name).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved master persistant disk", r, err)
		return false, err
	}
	cm.cluster.Spec.MasterDiskId = name
	return true, nil
}

func (cm *ClusterManager) createDisk(name, diskType string, sizeGb int64) (string, error) {
	// Type:        "https://www.googleapis.com/compute/v1/projects/tigerworks-kube/zones/us-central1-b/diskTypes/pd-ssd",
	// SourceImage: "https://www.googleapis.com/compute/v1/projects/google-containers/global/images/container-vm-v20150806",

	dType := fmt.Sprintf("projects/%v/zones/%v/diskTypes/%v", cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, diskType)

	r1, err := cm.conn.computeService.Disks.Insert(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, &compute.Disk{
		Name:   name,
		Zone:   cm.cluster.Spec.Cloud.Zone,
		Type:   dType,
		SizeGb: sizeGb,
	}).Do()

	Logger(cm.ctx).Debug("Created master disk", r1, err)
	if err != nil {
		return name, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.conn.waitForZoneOperation(r1.Name)
	if err != nil {
		return name, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Blank disk of type %v created before creating the master VM", dType)
	return name, nil
}

func (cm *ClusterManager) getReserveIP() (bool, error) {
	Logger(cm.ctx).Infof("Checking existence of reserved master ip ")
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		name := cm.namer.ReserveIPName()

		Logger(cm.ctx).Infof("Checking existence of reserved master ip %v", name)
		if r1, err := cm.conn.computeService.Addresses.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Region, name).Do(); err == nil {
			if r1.Status == "IN_USE" {
				return true, fmt.Errorf("Found a static IP with name %v in use. Failed to reserve a new ip with the same name.", name)
			}

			Logger(cm.ctx).Debug("Found master IP was already reserved", r1, err)
			cm.cluster.Spec.MasterReservedIP = r1.Address
			Logger(cm.ctx).Infof("Newly reserved master ip %v", cm.cluster.Spec.MasterReservedIP)
			return true, nil
		}
	}
	return false, nil

}

func (cm *ClusterManager) reserveIP() error {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		name := cm.namer.ReserveIPName()
		if _, err := cm.getReserveIP(); err != nil {
			return err
		}

		Logger(cm.ctx).Infof("Reserving master ip %v", name)
		r2, err := cm.conn.computeService.Addresses.Insert(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Region, &compute.Address{Name: name}).Do()
		Logger(cm.ctx).Debug("Reserved master IP", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		err = cm.conn.waitForRegionOperation(r2.Name)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		Logger(cm.ctx).Infof("Master ip %v reserved", name)
		if r3, err := cm.conn.computeService.Addresses.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Region, name).Do(); err == nil {
			Logger(cm.ctx).Debug("Retrieved newly reserved master IP", r3, err)

			cm.cluster.Spec.MasterReservedIP = r3.Address
			Logger(cm.ctx).Infof("Newly reserved master ip %v", cm.cluster.Spec.MasterReservedIP)
		}
	}

	return nil
}

func (cm *ClusterManager) getMasterInstance() (bool, error) {
	if r1, err := cm.conn.computeService.Instances.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, cm.cluster.Spec.KubernetesMasterName).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved master instance", r1, err)
		return false, err
	}
	return true, nil
}

func (cm *ClusterManager) createMasterIntance() (string, error) {
	// MachineType:  "projects/tigerworks-kube/zones/us-central1-b/machineTypes/n1-standard-1",
	// Zone:         "projects/tigerworks-kube/zones/us-central1-b",

	// startupScript := cm.RenderStartupScript(cm.cluster, cm.cluster.Spec.MasterSKU, api.RoleKubernetesMaster)
	startupScript, err := RenderStartupScript(cm.ctx, cm.cluster, api.RoleMaster, cm.cluster.Spec.MasterSKU)
	if err != nil {
		return "", err
	}

	machineType := fmt.Sprintf("projects/%v/zones/%v/machineTypes/%v", cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, cm.cluster.Spec.MasterSKU)
	zone := fmt.Sprintf("projects/%v/zones/%v", cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone)
	pdSrc := fmt.Sprintf("projects/%v/zones/%v/disks/%v", cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, cm.namer.MasterPDName())
	srcImage := fmt.Sprintf("projects/%v/global/images/%v", cm.cluster.Spec.Cloud.InstanceImageProject, cm.cluster.Spec.Cloud.InstanceImage)

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
				Network: fmt.Sprintf("projects/%v/global/networks/%v", cm.cluster.Spec.Cloud.Project, defaultNetwork),
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
	r1, err := cm.conn.computeService.Instances.Insert(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, instance).Do()
	Logger(cm.ctx).Debug("Created master instance", r1, err)
	if err != nil {
		return r1.Name, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Master instance of type %v in zone %v using persistent disk %v created", machineType, zone, pdSrc)
	return r1.Name, nil
}

// Instance
func (cm *ClusterManager) getInstance(instance string) (*api.Node, error) {
	Logger(cm.ctx).Infof("Retrieving instance %v in zone %v", instance, cm.cluster.Spec.Cloud.Zone)
	r1, err := cm.conn.computeService.Instances.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, instance).Do()
	Logger(cm.ctx).Debug("Retrieved instance", r1, err)
	if err != nil {
		return nil, err
	}
	return cm.newKubeInstance(r1)
}

func (cm *ClusterManager) listInstances(instanceGroup string) ([]*api.Node, error) {
	Logger(cm.ctx).Infof("Retrieving instances in node group %v", instanceGroup)
	instances := make([]*api.Node, 0)
	r1, err := cm.conn.computeService.InstanceGroups.ListInstances(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, instanceGroup, &compute.InstanceGroupsListInstancesRequest{
		InstanceState: "ALL",
	}).Do()
	Logger(cm.ctx).Debug("Retrieved instance", r1, err)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	for _, item := range r1.Items {
		name := item.Instance[strings.LastIndex(item.Instance, "/")+1:]
		r2, err := cm.conn.computeService.Instances.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, name).Do()
		if err != nil {
			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		instance, err := cm.newKubeInstance(r2)
		if err != nil {
			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		instance.Spec.Role = api.RoleNode
		instances = append(instances, instance)
	}
	return instances, nil
}

func (cm *ClusterManager) newKubeInstance(r1 *compute.Instance) (*api.Node, error) {
	for _, accessConfig := range r1.NetworkInterfaces[0].AccessConfigs {
		if accessConfig.Type == "ONE_TO_ONE_NAT" {
			i := api.Node{
				ObjectMeta: metav1.ObjectMeta{
					UID:  phid.NewKubeInstance(),
					Name: r1.Name,
				},
				Spec: api.NodeSpec{
					SKU: r1.MachineType[strings.LastIndex(r1.MachineType, "/")+1:],
				},
				Status: api.NodeStatus{
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
				i.Status.Phase = api.NodeDeleted
			} else {
				i.Status.Phase = api.NodeReady
			}
			return &i, nil
		}
	}
	return nil, errors.New("Failed to convert gcloud instance to KubeInstance.").WithContext(cm.ctx).Err() //stackerr.New("Failed to convert gcloud instance to KubeInstance.")
}

func (cm *ClusterManager) getNodeFirewallRule() (bool, error) {
	name := cm.cluster.Name + "-node-all"
	if r1, err := cm.conn.computeService.Firewalls.Get(cm.cluster.Spec.Cloud.Project, name).Do(); err != nil {
		Logger(cm.ctx).Debug("Retrieved node firewall rule", r1, err)
		return false, err
	}
	return true, nil
}

func (cm *ClusterManager) createNodeFirewallRule() (string, error) {
	name := cm.cluster.Name + "-node-all"
	network := fmt.Sprintf("projects/%v/global/networks/%v", cm.cluster.Spec.Cloud.Project, defaultNetwork)

	r1, err := cm.conn.computeService.Firewalls.Insert(cm.cluster.Spec.Cloud.Project, &compute.Firewall{
		Name:         name,
		Network:      network,
		SourceRanges: []string{cm.cluster.Spec.Networking.PodSubnet},
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
	Logger(cm.ctx).Debug("Created firewall rule", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Node firewall rule %v created", name)
	return r1.Name, nil
}

func (cm *ClusterManager) createNodeInstanceTemplate(sku string) (string, error) {
	templateName := cm.namer.InstanceTemplateName(sku)

	Logger(cm.ctx).Infof("Retrieving node template %v", templateName)
	if r1, err := cm.conn.computeService.InstanceTemplates.Get(cm.cluster.Spec.Cloud.Project, templateName).Do(); err == nil {
		Logger(cm.ctx).Debug("Retrieved node template", r1, err)

		Logger(cm.ctx).Infof("Deleting node template %v", templateName)
		if r2, err := cm.conn.computeService.InstanceTemplates.Delete(cm.cluster.Spec.Cloud.Project, templateName).Do(); err != nil {
			Logger(cm.ctx).Debug("Delete node template called", r2, err)
			Logger(cm.ctx).Infoln("Failed to delete existing instance template")
			os.Exit(1)
		}
	}
	//  if cluster.Spec.ctx.Preemptiblenode == "true" {
	//	  preemptible_nodes = "--preemptible --maintenance-policy TERMINATE"
	//  }

	startupScript, err := RenderStartupScript(cm.ctx, cm.cluster, api.RoleNode, sku)
	if err != nil {
		return "", err
	}

	image := fmt.Sprintf("projects/%v/global/images/%v", cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.InstanceImage)
	network := fmt.Sprintf("projects/%v/global/networks/%v", cm.cluster.Spec.Cloud.Project, defaultNetwork)

	Logger(cm.ctx).Infof("Create instance template %v", templateName)
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
	r1, err := cm.conn.computeService.InstanceTemplates.Insert(cm.cluster.Spec.Cloud.Project, tpl).Do()
	Logger(cm.ctx).Debug("Create instance template called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Node instance template %v created for sku %v", templateName, sku)
	return r1.Name, nil
}

func (cm *ClusterManager) createNodeGroup(sku string, count int64) (string, error) {
	name := cm.namer.NodeGroupName(sku)
	template := fmt.Sprintf("projects/%v/global/instanceTemplates/%v", cm.cluster.Spec.Cloud.Project, cm.namer.InstanceTemplateName(sku))

	Logger(cm.ctx).Infof("Creating instance group %v from template %v", name, template)
	r1, err := cm.conn.computeService.InstanceGroupManagers.Insert(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, &compute.InstanceGroupManager{
		Name:             name,
		BaseInstanceName: cm.cluster.Name + "-node-" + sku,
		TargetSize:       count,
		InstanceTemplate: template,
	}).Do()
	Logger(cm.ctx).Debug("Create instance group called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Instance group %v created with %v nodes of %v sku", name, count, sku)
	return r1.Name, nil
}

// Not used since Kube 1.3
func (cm *ClusterManager) createAutoscaler(sku string, count int64) (string, error) {
	name := cm.namer.NodeGroupName(sku)
	target := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%v/zones/%v/instanceGroupManagers/%v", cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, name)

	Logger(cm.ctx).Infof("Creating auto scaler %v for instance group %v", name, target)
	r1, err := cm.conn.computeService.Autoscalers.Insert(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, &compute.Autoscaler{
		Name:   name,
		Target: target,
		AutoscalingPolicy: &compute.AutoscalingPolicy{
			MinNumReplicas: int64(1),
			MaxNumReplicas: count,
		},
	}).Do()
	Logger(cm.ctx).Debug("Create auto scaler called", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Auto scaler %v for instance group %v created", name, target)
	return r1.Name, nil
}

func (cm *ClusterManager) GetInstance(md *api.NodeStatus) (*api.Node, error) {
	r2, err := cm.conn.computeService.Instances.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, md.Name).Do()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	i, err := cm.newKubeInstance(r2)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return i, nil
}
