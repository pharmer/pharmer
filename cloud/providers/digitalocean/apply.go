package digitalocean

import (
	//"context"
	"fmt"
	"strconv"

	api "github.com/pharmer/pharmer/apis/v1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error
	var acts []api.Action

	if in.Status.Phase == "" {
		return nil, errors.Errorf("cluster `%s` is in unknown phase", cm.cluster.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if err = cm.PrepareCloud(in.Name); err != nil {
		return nil, err
	}
	if cm.cluster.Status.Phase == api.ClusterUpgrading {
		return nil, errors.Errorf("cluster `%s` is upgrading. Retry after cluster returns to Ready state", cm.cluster.Name)
	}
	if cm.cluster.Status.Phase == api.ClusterReady {
		var kc kubernetes.Interface
		kc, err = cm.GetAdminClient()
		if err != nil {
			return nil, err
		}
		if upgrade, err := NewKubeVersionGetter(kc, cm.cluster).IsUpgradeRequested(); err != nil {
			return nil, err
		} else if upgrade {
			cm.cluster.Status.Phase = api.ClusterUpgrading
			Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
			return cm.applyUpgrade(dryRun)
		}
	}

	if cm.cluster.Status.Phase == api.ClusterPending {
		a, err := cm.applyCreate(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		nodeGroups, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, ng := range nodeGroups {
			ng.Spec.Nodes = 0
			_, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).Update(ng)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		a, err := cm.applyScale(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		a, err := cm.applyDelete(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}
	return acts, nil
}

// Creates network, and creates ready master(s)
func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	var found bool
	found, _, err = cm.conn.getPublicKey()
	if err != nil {
		return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "PublicKey",
			Message:  "Public key will be imported",
		})
		if !dryRun {
			cm.cluster.Status.Cloud.SShKeyExternalID, err = cm.conn.importPublicKey()
			if err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "PublicKey",
			Message:  "Public key found",
		})
	}

	// ignore errors, since tags are simply informational.
	found, err = cm.conn.getTags()
	if err != nil {
		return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Tag",
			Message:  fmt.Sprintf("Tag %s will be added", "KubernetesCluster:"+cm.cluster.Name),
		})
		if !dryRun {
			cm.conn.createTags()
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Tag",
			Message:  fmt.Sprintf("Tag %s found", "KubernetesCluster:"+cm.cluster.Name),
		})
	}

	// -------------------------------------------------------------------ASSETS

	var masterNG []*clusterv1.Machine
	masterNG, err = FindMasterMachines(cm.cluster)
	if err != nil {
		return
	}
	var lbIp string
	haSetup := IsHASetup(cm.cluster)
	if haSetup {
		Logger(cm.ctx).Info("Creating loadbalancer")
		lbIp, err = cm.conn.createLoadBalancer(cm.ctx, cm.namer.LoadBalancerName())
		if err != nil {
			return
		}
		Logger(cm.ctx).Infof("Created loadbalancer lbIp = %v", lbIp)
	}
	etcdServerAddress := ""
	for _, master := range masterNG {
		if d, _ := cm.conn.instanceIfExists(master); d == nil {
			Logger(cm.ctx).Info("Creating master instance")
			acts = append(acts, api.Action{
				Action:   api.ActionAdd,
				Resource: "MasterInstance",
				Message:  fmt.Sprintf("Master instance %s will be created", cm.namer.MasterName()),
			})
			if !dryRun {
				if haSetup {
					master.Labels[api.PharmerHASetup] = "true"
					master.Labels[api.PharmerLoadBalancerIP] = lbIp
				}
				var masterServer *api.NodeInfo
				if etcdServerAddress != "" {
					master.Labels[api.EtcdServerAddress] = etcdServerAddress
				}
				masterServer, err = cm.conn.CreateInstance(cm.cluster, master, "")
				if err != nil {
					return
				}
				if masterServer.PrivateIP != "" {
					cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
						Type:    core.NodeInternalIP,
						Address: masterServer.PrivateIP,
					})
				}
				if !haSetup {
					var providerConf *api.MachineProviderConfig
					providerConf, err = cm.cluster.MachineProviderConfig(master)
					if err != nil {
						return
					}
					if providerConf.Config.ExternalIPType == api.IPTypeReserved {
						if len(cm.cluster.Status.ReservedIPs) == 0 {
							acts = append(acts, api.Action{
								Action:   api.ActionAdd,
								Resource: "ReserveIP",
								Message:  "ReservedIP will be created",
							})
							if !dryRun {
								var reservedIP string
								reservedIP, err = cm.conn.createReserveIP()
								if err != nil {
									return
								}
								id, _ := strconv.Atoi(masterServer.ExternalID)
								if err = cm.conn.assignReservedIP(reservedIP, id); err != nil {
									return
								}
								cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
									Type:    core.NodeExternalIP,
									Address: reservedIP,
								})
								cm.cluster.Status.ReservedIPs = append(cm.cluster.Status.ReservedIPs, api.ReservedIP{
									IP: reservedIP,
								})
							}
						} else {
							reservedIP := cm.cluster.Status.ReservedIPs[0].IP
							found, err = cm.conn.getReserveIP(reservedIP)
							if err != nil {
								return
							}
							if !found {
								err = errors.Errorf("ReservedIP %s not found", reservedIP)
								return
							} else {
								acts = append(acts, api.Action{
									Action:   api.ActionNOP,
									Resource: "ReserveIP",
									Message:  fmt.Sprintf("Reserved lbIp %s found", reservedIP),
								})
							}
							if reservedIP != masterServer.PublicIP {
								id, _ := strconv.Atoi(masterServer.ExternalID)
								if err = cm.conn.assignReservedIP(reservedIP, id); err != nil {
									return
								}
								cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
									Type:    core.NodeExternalIP,
									Address: reservedIP,
								})
								cm.cluster.Status.ReservedIPs = append(cm.cluster.Status.ReservedIPs, api.ReservedIP{
									IP: reservedIP,
								})
							}
						}
					} else {
						if masterServer.PublicIP != "" {
							cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
								Type:    core.NodeExternalIP,
								Address: masterServer.PublicIP,
							})
						}
					}
					err = cm.cluster.SetClusterApiEndpoints()
					if err != nil {
						return
					}
				} else {
					var did int
					did, err = strconv.Atoi(masterServer.ExternalID)
					if err != nil {
						return
					}
					if err = cm.conn.addNodeToBalancer(cm.ctx, cm.namer.LoadBalancerName(), did); err != nil {
						return
					}
					cm.cluster.Spec.ClusterAPI.Status.APIEndpoints = []clusterv1.APIEndpoint{
						{
							Host: masterServer.PublicIP,
							Port: int(cm.cluster.Spec.API.BindPort),
						},
					}
					/*append(cm.cluster.Spec.ClusterAPI.Status.APIEndpoints, clusterv1.APIEndpoint{
						Host: masterServer.PublicIP,
						Port: int(cm.cluster.Spec.API.BindPort),
					})*/
					//	Store(cm.ctx).Clusters().Update(cm.cluster)
				}
				if etcdServerAddress == "" {
					var kc kubernetes.Interface
					kc, err = cm.GetAdminClient()
					if err != nil {
						return
					}
					// wait for nodes to start
					if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
						return
					}
					etcdServerAddress = masterServer.PublicIP
				}

			}
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "MasterInstance",
				Message:  fmt.Sprintf("master instance %v already exist", master.Name),
			})
		}
	}

	if haSetup {
		cm.cluster.Spec.ClusterAPI.Status.APIEndpoints = []clusterv1.APIEndpoint{
			{
				Host: lbIp,
				Port: int(cm.cluster.Spec.API.BindPort),
			},
		}
	}

	Store(cm.ctx).Clusters().Update(cm.cluster)

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}
	// wait for nodes to start
	if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
		return
	}

	// needed to get master_internal_ip
	cm.cluster.Status.Phase = api.ClusterReady
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		return
	}

	// need to run ccm
	if err = CreateCredentialSecret(cm.ctx, kc, cm.cluster); err != nil {
		return
	}

	ca, err := NewClusterApi(cm.ctx, cm.cluster, kc)
	if err != nil {
		return acts, err
	}
	if err := ca.Apply(); err != nil {
		return acts, err
	}
	return acts, err
}

// Scales up/down regular node groups
func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	Logger(cm.ctx).Infoln("scaling machine set")
	var cs clientset.Interface
	cs, err = NewClusterApiClient(cm.ctx, cm.cluster)
	if err != nil {
		return
	}
	client := cs.ClusterV1alpha1()

	var machineSet []*clusterv1.MachineSet
	//var msc *clusterv1.MachineSet
	machineSet, err = Store(cm.ctx).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, ms := range machineSet {
		if ms.DeletionTimestamp != nil {
			if err = client.MachineSets(core.NamespaceDefault).Delete(ms.Name, &metav1.DeleteOptions{}); err != nil {
				return
			}
			err = Store(cm.ctx).MachineSet(cm.cluster.Name).Delete(ms.Name)
			return
		}

		_, err = client.MachineSets(core.NamespaceDefault).Get(ms.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = client.MachineSets(core.NamespaceDefault).Create(ms)
			if err != nil {
				return
			}
		} else {
			if _, err = client.MachineSets(core.NamespaceDefault).Update(ms); err != nil {
				return
			}

			//patch makes provider config null :(. TODO(): why??
			/*if _, err = PatchMachineSet(cs, msc, ms); err != nil {
				return
			}*/
		}

	}

	return
}

// Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	Logger(cm.ctx).Infoln("deleting cluster")
	var found bool

	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}
	var masterInstance *core.Node
	masterInstance, err = kc.CoreV1().Nodes().Get(cm.namer.MasterName(), metav1.GetOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		Logger(cm.ctx).Infof("master instance not found. Reason: %s", masterInstance.Spec.ProviderID, err)
	} else if err == nil {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Will delete master instance with name %v", cm.namer.MasterName()),
		})
		if !dryRun {
			err = cm.conn.DeleteInstanceByProviderID(masterInstance.Spec.ProviderID)
			if err != nil {
				Logger(cm.ctx).Infof("Failed to delete instance %s. Reason: %s", masterInstance.Spec.ProviderID, err)
			}
			/*if masterNG.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
				for _, addr := range masterInstance.Status.Addresses {
					if addr.Type == core.NodeExternalIP {
						err = cm.conn.releaseReservedIP(addr.Address)
						if err != nil {
							return
						}
					}
				}
			}*/
		}
	}

	// delete by tag
	tag := "KubernetesCluster:" + cm.cluster.Name
	_, err = cm.conn.client.Droplets.DeleteByTag(cm.ctx, tag)
	if err != nil {
		Logger(cm.ctx).Infof("Failed to delete resources by tag %s. Reason: %s", tag, err)
	}
	Logger(cm.ctx).Infof("Deleted droplet by tag %s", tag)

	// Delete SSH key
	found, _, err = cm.conn.getPublicKey()
	if err != nil {
		return
	}
	if found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "PublicKey",
			Message:  "Public key will be deleted",
		})
		if !dryRun {
			err = cm.conn.deleteSSHKey()
			if err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "PublicKey",
			Message:  "Public key not found",
		})
	}
	if IsHASetup(cm.cluster) {
		cm.conn.deleteLoadBalancer(cm.ctx, cm.namer.LoadBalancerName())
	}

	// Failed
	cm.cluster.Status.Phase = api.ClusterDeleted
	_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	Logger(cm.ctx).Infof("Cluster %v deletion is deleted successfully", cm.cluster.Name)
	return
}

func (cm *ClusterManager) applyUpgrade(dryRun bool) (acts []api.Action, err error) {
	var kc kubernetes.Interface
	if kc, err = cm.GetAdminClient(); err != nil {
		return
	}

	upm := NewUpgradeManager(cm.ctx, cm, kc, cm.cluster)
	a, err := upm.Apply(dryRun)
	if err != nil {
		return
	}
	acts = append(acts, a...)
	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}
	return
}
