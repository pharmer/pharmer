package digitalocean

import (
	"context"
	"fmt"
	"strconv"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	apiv1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error
	var acts []api.Action

	if in.Status.Phase == "" {
		return nil, fmt.Errorf("cluster `%s` is in unknown phase", cm.cluster.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
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

// Creates network, and creates a ready master
func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	var found bool
	found, err = cm.conn.getPublicKey()
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
			err = cm.conn.importPublicKey()
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
	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	masterNG := FindMasterNodeGroup(nodeGroups)
	if masterNG.Spec.Template.Spec.SKU == "" {
		masterNG.Spec.Template.Spec.SKU = "2gb"
		masterNG, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).Update(masterNG)
		if err != nil {
			return
		}
	}

	if masterNG.Status.Nodes < masterNG.Spec.Nodes {
		Logger(cm.ctx).Info("Creating master instance")
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Master instance %s will be created", cm.namer.MasterName()),
		})
		if !dryRun {
			var masterDroplet *api.SimpleNode
			masterDroplet, err = cm.conn.CreateInstance(cm.namer.MasterName(), masterNG)
			if err != nil {
				return
			}
			if masterDroplet.PublicIP != "" {
				cm.cluster.Status.APIAddress = append(cm.cluster.Status.APIAddress, api.Address{
					Type: api.AddressTypeExternalIP,
					Host: masterDroplet.PublicIP,
				})
			}
			if masterDroplet.PrivateIP != "" {
				cm.cluster.Status.APIAddress = append(cm.cluster.Status.APIAddress, api.Address{
					Type: api.AddressTypeInternalIP,
					Host: masterDroplet.PrivateIP,
				})
			}

			if masterNG.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
				var masterNIP *api.NodeIP
				for _, rip := range masterNG.Status.ExternalIPs {
					if rip.Name == cm.namer.MasterName() {
						masterNIP = &rip
						break
					}
				}
				if masterNIP == nil {
					acts = append(acts, api.Action{
						Action:   api.ActionAdd,
						Resource: "ReserveIP",
						Message:  "ReservedIP will be created",
					})
					if !dryRun {
						masterNIP = &api.NodeIP{
							Name: cm.namer.MasterName(),
						}
						masterNIP.IP, err = cm.conn.createReserveIP()
						if err != nil {
							return
						}
						masterNG.Status.ExternalIPs = append(masterNG.Status.ExternalIPs, *masterNIP)
					}
				} else {
					found, err = cm.conn.getReserveIP(masterNIP.IP)
					if err != nil {
						return
					}
					if !found {
						err = fmt.Errorf("ReservedIP %s not found", masterNIP.IP)
						return
					} else {
						acts = append(acts, api.Action{
							Action:   api.ActionNOP,
							Resource: "ReserveIP",
							Message:  fmt.Sprintf("Reserved ip %s found", masterNIP.IP),
						})
					}
				}
				id, _ := strconv.Atoi(masterDroplet.ExternalID)
				if err = cm.conn.assignReservedIP(masterNIP.IP, id); err != nil {
					return
				}
				cm.cluster.Status.APIAddress = append(cm.cluster.Status.APIAddress, api.Address{
					Type: api.AddressTypeReservedIP,
					Host: masterNIP.IP,
				})
			}

			var kc kubernetes.Interface
			kc, err = cm.GetAdminClient()
			if err != nil {
				return
			}
			// wait for nodes to start
			if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
				return
			}

			masterNG.Status.Nodes = 1
			masterNG, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(masterNG)
			if err != nil {
				return
			}

			err = EnsureARecord2(cm.ctx, cm.cluster, masterDroplet.PublicIP, masterDroplet.PrivateIP) // works for reserved or non-reserved mode
			if err != nil {
				return
			}
			// Wait for master A record to propagate
			if err = EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
				return
			}
			// needed to get master_internal_ip
			cm.cluster.Status.Phase = api.ClusterReady
			if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "MasterInstance",
			Message:  "master instance(s) already exist",
		})
	}

	return
}

// Scales up/down regular nodes
func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var kc kubernetes.Interface
	if cm.cluster.Status.Phase != api.ClusterPending {
		kc, err = cm.GetAdminClient()
		if err != nil {
			return
		}
	}
	for _, ng := range nodeGroups {
		if ng.IsMaster() {
			continue
		}
		igm := NewNodeGroupManager(ng, cm.conn, kc)
		var a2 []api.Action
		a2, err = igm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a2...)
	}
	return
}

func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	var found bool

	if cm.cluster.Status.Phase == api.ClusterPending {
		cm.cluster.Status.Phase = api.ClusterFailing
	} else if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	masterNG := FindMasterNodeGroup(nodeGroups)

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}
	var masterInstance *apiv1.Node
	masterInstance, err = kc.CoreV1().Nodes().Get(cm.namer.MasterName(), metav1.GetOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return
	} else if err == nil {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Will delete master instance with name %v", cm.cluster.Spec.KubernetesMasterName),
		})
		if !dryRun {
			err = cm.conn.DeleteInstanceByProviderID(masterInstance.Spec.ProviderID)
			if err != nil {
				return
			}
			if masterNG.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
				var masterNIP *api.NodeIP
				for _, rip := range masterNG.Status.ExternalIPs {
					if rip.Name == cm.namer.MasterName() {
						masterNIP = &rip
						break
					}
				}
				err = cm.conn.releaseReservedIP(masterNIP.IP)
				if err != nil {
					return
				}
			}
		}
	}

	// delete by tag
	_, err = cm.conn.client.Droplets.DeleteByTag(context.TODO(), "KubernetesCluster:"+cm.cluster.Name)
	if err != nil {
		return
	}
	Logger(cm.ctx).Infof("Deleted droplet by tag %v", "KubernetesCluster:"+cm.cluster.Name)

	// Delete SSH key
	found, err = cm.conn.getPublicKey()
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

	if !dryRun {
		if err = DeleteARecords(cm.ctx, cm.cluster); err != nil {
			return
		}
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
