package gce

import (
	"fmt"

	. "github.com/appscode/go/context"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}
	cm.conn.namer = cm.namer

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
		nodeGroups, err := Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, ng := range nodeGroups {
			ng.Spec.Nodes = 0
			_, err := Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).Update(ng)
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

func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	var found bool
	if !dryRun {
		if err = cm.conn.importPublicKey(); err != nil {
			return
		}
	}

	// TODO: Should we add *IfMissing suffix to all these functions
	found, _ = cm.conn.getNetworks()

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Default Network",
			Message:  "Not found, will add default network with ipv4 range 10.240.0.0/16",
		})
		if !dryRun {
			if err = cm.conn.ensureNetworks(); err != nil {
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

	found, _ = cm.conn.getFirewallRules()
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Default Firewall rule",
			Message:  "default-allow-internal, default-allow-ssh, https rules will be created",
		})
		if !dryRun {
			if err = cm.conn.ensureFirewallRules(); err != nil {
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

	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, ID(cm.ctx))
		return
	}
	var masterNG *api.NodeGroup
	masterNG, err = FindMasterNodeGroup(nodeGroups)
	if err != nil {
		return
	}
	if masterNG.Spec.Template.Spec.SKU == "" {
		totalNodes := NodeCount(nodeGroups)
		masterNG.Spec.Template.Spec.SKU = "n1-standard-2"
		if totalNodes > 10 {
			masterNG.Spec.Template.Spec.SKU = "n1-standard-4"
		}
		if totalNodes > 100 {
			masterNG.Spec.Template.Spec.SKU = "n1-standard-8"
		}
		if totalNodes > 250 {
			masterNG.Spec.Template.Spec.SKU = "n1-standard-16"
		}
		if totalNodes > 500 {
			masterNG.Spec.Template.Spec.SKU = "n1-standard-32"
		}
		masterNG, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).Update(masterNG)
		if err != nil {
			return
		}
	}
	found, _ = cm.conn.getMasterPDDisk(cm.namer.MasterPDName())
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master persistent disk",
			Message:  fmt.Sprintf("Not found, will be added with disk type %v, size %v and name %v", masterNG.Spec.Template.Spec.DiskType, masterNG.Spec.Template.Spec.DiskSize, cm.namer.MasterPDName()),
		})
		if !dryRun {
			cm.cluster.Spec.MasterDiskId, err = cm.conn.createDisk(cm.namer.MasterPDName(), masterNG.Spec.Template.Spec.DiskType, masterNG.Spec.Template.Spec.DiskSize)
			if err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master persistent disk",
			Message:  fmt.Sprintf("Found master persistent disk with disk type %v, size %v and name %v", masterNG.Spec.Template.Spec.DiskType, masterNG.Spec.Template.Spec.DiskSize, cm.namer.MasterPDName()),
		})

	}

	var reservedIP string
	if masterNG.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
		if len(cm.cluster.Status.ReservedIPs) == 0 {
			acts = append(acts, api.Action{
				Action:   api.ActionAdd,
				Resource: "ReserveIP",
				Message:  "ReservedIP will be created",
			})
			if !dryRun {
				if reservedIP, err = cm.conn.reserveIP(); err != nil {
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
			reservedIP = cm.cluster.Status.ReservedIPs[0].IP
			found, err = cm.conn.getReserveIP()
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
					Message:  fmt.Sprintf("Reserved ip %s found", reservedIP),
				})
			}
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "Reserve IP",
				Message:  fmt.Sprintf("Found, MasterReservedIP = %s", reservedIP),
			})
		}
	}

	// needed for master start-up config
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.Wrap(err, ID(cm.ctx))
		return
	}

	found, _ = cm.conn.getMasterInstance()
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Master instance with name %v will be created", cm.namer.MasterName()),
		})
		if !dryRun {
			var op1 string
			if op1, err = cm.conn.createMasterIntance(masterNG); err != nil {
				return
			}

			if err = cm.conn.waitForZoneOperation(op1); err != nil {
				return
			}

			var masterInstance *api.NodeInfo
			masterInstance, err = cm.conn.getInstance(cm.namer.MasterName())
			if err != nil {
				return acts, err
			}

			Logger(cm.ctx).Info("Waiting for cluster initialization")

			if masterInstance.PrivateIP != "" {
				cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
					Type:    core.NodeInternalIP,
					Address: masterInstance.PrivateIP,
				})
			}
			if masterNG.Spec.Template.Spec.ExternalIPType != api.IPTypeReserved {
				cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
					Type:    core.NodeExternalIP,
					Address: masterInstance.PublicIP,
				})
			}

			// wait for nodes to start
			var kc kubernetes.Interface
			kc, err = cm.GetAdminClient()
			if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, ID(cm.ctx))
				return acts, err
			}

			masterNG.Status.Nodes = 1
			masterNG, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).UpdateStatus(masterNG)
			if err != nil {
				return
			}
			// -------------------------------------------------------------------------------------------------------------
			// needed to get master_internal_ip
			cm.cluster.Status.Phase = api.ClusterReady
			if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
				return
			}
		}

	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  "master instance(s) already exist",
		})
	}

	return
}

func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	var token string
	var kc kubernetes.Interface
	if cm.cluster.Status.Phase != api.ClusterPending {
		kc, err = cm.GetAdminClient()
		if err != nil {
			return
		}
		if !dryRun {
			if token, err = GetExistingKubeadmToken(kc, api.TokenDuration_10yr); err != nil {
				return
			}
		}
	}

	// needed for node start-up config to get master_internal_ip
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.Wrap(err, ID(cm.ctx))
		return
	}

	if found, _ := cm.conn.getNodeFirewallRule(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Node Firewall Rule",
			Message:  fmt.Sprintf("%v node firewall rule will be created", cm.cluster.Name+"-node-all"),
		})
		if !dryRun {
			// Use zone operation to wait and block.
			if op2, err := cm.conn.createNodeFirewallRule(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.Wrap(err, ID(cm.ctx))
				return acts, err
			} else {
				if err = cm.conn.waitForGlobalOperation(op2); err != nil {
					cm.cluster.Status.Reason = err.Error()
					err = errors.Wrap(err, ID(cm.ctx))
					return acts, err
				}
			}
		}
	}

	for _, node := range nodeGroups {
		if node.IsMaster() {
			continue
		}
		igm := NewGCENodeGroupManager(cm.ctx, cm.conn, cm.namer, node, kc, token)
		var a2 []api.Action
		a2, err = igm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a2...)
	}

	Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	Store(cm.ctx).Clusters().Update(cm.cluster)

	return
}

func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var masterNG *api.NodeGroup
	masterNG, err = FindMasterNodeGroup(nodeGroups)
	if err != nil {
		return
	}
	for _, ng := range nodeGroups {
		if !ng.IsMaster() {
			template := cm.namer.InstanceTemplateName(ng.Spec.Template.Spec.SKU)
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Node Group",
				Message:  fmt.Sprintf("%v node group with template %v will be deleted", ng.Name, template),
			})
			if !dryRun {
				if err = cm.conn.deleteOnlyNodeGroup(ng.Name, template); err != nil {
					Logger(cm.ctx).Infof("Error on deleting node group. Reason: %v", err)
				}
				Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).Delete(ng.Name)
			}
		}
	}
	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Master Instance",
		Message:  fmt.Sprintf("Found master instance with name %v", cm.namer.MasterName()),
	})
	if !dryRun {
		if err = cm.conn.deleteMaster(); err != nil {
			Logger(cm.ctx).Infof("Error on deleting master. Reason: %v", err)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Node Firewall Rule",
		Message:  fmt.Sprintf("%v node firewall rule will be deleted", cm.cluster.Name+"-node-all"),
	})
	if !dryRun {
		if err = cm.conn.deleteFirewalls(); err != nil {
			cm.cluster.Status.Reason = err.Error()
		}
	}

	if masterNG.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
		if !dryRun {
			if err = cm.conn.releaseReservedIP(); err != nil {
				Logger(cm.ctx).Infof("Error on releasing reserve ip. Reason: %v", err)
			}
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Master persistent disk",
		Message:  fmt.Sprintf("Will delete master persistent with name %v", cm.namer.MasterPDName()),
	})

	if !dryRun {
		if err = cm.conn.deleteDisk(); err != nil {
			Logger(cm.ctx).Infof("Error on deleting disk. Reason: %v", err)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Route",
		Message:  fmt.Sprintf("Route will be delete"),
	})
	if !dryRun {
		if err = cm.conn.deleteRoutes(); err != nil {
			Logger(cm.ctx).Infof("Error on deleting routes. Reason: %v", err)
		}
	}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterDeleted
		Store(cm.ctx).Clusters().Update(cm.cluster)
	}

	return
}

func (cm *ClusterManager) applyUpgrade(dryRun bool) (acts []api.Action, err error) {
	var kc kubernetes.Interface
	if kc, err = cm.GetAdminClient(); err != nil {
		return
	}

	upm := NewUpgradeManager(cm.ctx, cm, kc, cm.cluster, cm.owner)
	if !dryRun {
		var a []api.Action
		a, err = upm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a...)
	}

	var nodeGroups []*api.NodeGroup
	if nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{}); err != nil {
		return
	}

	var token string
	if !dryRun {
		if token, err = GetExistingKubeadmToken(kc, api.TokenDuration_10yr); err != nil {
			return
		}
		if cm.cluster, err = Store(cm.ctx).Clusters().Update(cm.cluster); err != nil {
			return
		}
	}

	for _, ng := range nodeGroups {
		if !ng.IsMaster() {
			acts = append(acts, api.Action{
				Action:   api.ActionUpdate,
				Resource: "Instance Template",
				Message:  fmt.Sprintf("Instance template of %v will be updated to %v", ng.Name, cm.namer.InstanceTemplateName(ng.Spec.Template.Spec.SKU)),
			})
			if !dryRun {
				if err = cm.conn.updateNodeGroupTemplate(ng, token); err != nil {
					return
				}
			}
		}
	}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}

	return
}
