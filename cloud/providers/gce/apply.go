package gce

import (
	"fmt"
	"time"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	apiv1 "k8s.io/api/core/v1"
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
	cm.conn.namer = cm.namer

	if cm.cluster.Status.Phase == api.ClusterUpgrading {
		return cm.applyUpgrade(dryRun)
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
	nodeGroups, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}
	masterNG := FindMasterNodeGroup(nodeGroups)

	if masterNG.Spec.Template.Spec.SKU == "" {
		totalNodes := NodeCount(nodeGroups)
		masterNG.Spec.Template.Spec.SKU = "n1-standard-1"
		if totalNodes > 5 {
			masterNG.Spec.Template.Spec.SKU = "n1-standard-2"
		}
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
	}
	found, _ = cm.conn.getMasterPDDisk(cm.namer.MasterPDName())
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master persistant disk",
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
			Resource: "Master persistant disk",
			Message:  fmt.Sprintf("Found master persistant disk with disk type %v, size %v and name %v", masterNG.Spec.Template.Spec.DiskType, masterNG.Spec.Template.Spec.DiskSize, cm.namer.MasterPDName()),
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
				cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, apiv1.NodeAddress{
					Type:    apiv1.NodeExternalIP,
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
				err = fmt.Errorf("ReservedIP %s not found", reservedIP)
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
				Message:  fmt.Sprintf("Found, MasterReservedIP = ", cm.cluster.Spec.MasterReservedIP),
			})
		}
	}

	// needed for master start-up config
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	found, _ = cm.conn.getMasterInstance()
	if !found {
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
		if !dryRun {
			var op1 string
			if op1, err = cm.conn.createMasterIntance(masterNG); err != nil {
				return
			}

			if err = cm.conn.waitForZoneOperation(op1); err != nil {
				return
			}

			var masterInstance *api.SimpleNode
			masterInstance, err = cm.conn.getInstance(cm.namer.MasterName())
			if err != nil {
				return acts, err
			}

			err = EnsureARecord2(cm.ctx, cm.cluster, masterInstance.PublicIP, masterInstance.PrivateIP) // works for reserved or non-reserved mode
			if err != nil {
				return
			}

			Logger(cm.ctx).Info("Waiting for cluster initialization")

			// Wait for master A record to propagate
			if err = EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return acts, err
			}

			if masterInstance.PrivateIP != "" {
				cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, apiv1.NodeAddress{
					Type:    apiv1.NodeInternalIP,
					Address: masterInstance.PrivateIP,
				})
			}
			if masterNG.Spec.Template.Spec.ExternalIPType != api.IPTypeReserved {
				cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, apiv1.NodeAddress{
					Type:    apiv1.NodeExternalIP,
					Address: masterInstance.PublicIP,
				})
			}

			// wait for nodes to start
			var kc kubernetes.Interface
			kc, err = cm.GetAdminClient()
			if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return acts, err
			}

			masterNG.Status.Nodes = 1
			masterNG, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(masterNG)
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
		if !dryRun {
			if cm.cluster.Spec.Token, err = GetExistingKubeadmToken(kc); err != nil {
				return
			}
			if cm.cluster, err = Store(cm.ctx).Clusters().Update(cm.cluster); err != nil {
				return
			}
		}
	}

	// needed for node start-up config to get master_internal_ip
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.FromErr(err).WithContext(cm.ctx).Err()
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
	}

	for _, node := range nodeGroups {
		if node.IsMaster() {
			continue
		}
		igm := NewGCENodeGroupManager(cm.ctx, cm.conn, cm.namer, node, kc)
		var a2 []api.Action
		a2, err = igm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a2...)
	}

	if !dryRun && cm.cluster.Status.Phase == api.ClusterReady {
		time.Sleep(1 * time.Minute)

		for _, ng := range nodeGroups {
			if ng.IsMaster() {
				continue
			}
			providerInstances, _ := cm.conn.listInstances(ng.Name)
			fmt.Println(providerInstances)
			runningInstance := make(map[string]*api.SimpleNode)
			for _, node := range providerInstances {
				runningInstance[node.Name] = node
			}

			clusterInstance, _ := GetClusterIstance2(kc, ng.Name)
			fmt.Println(clusterInstance)
			for _, node := range clusterInstance {
				if _, found := runningInstance[node]; !found {
					if err = DeleteClusterInstance2(kc, node); err != nil {
						return
					}

				}
			}
		}
		Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		Store(cm.ctx).Clusters().Update(cm.cluster)
	}
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
	nodeGroups, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	masterNG := FindMasterNodeGroup(nodeGroups)

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
					//return
				}
				Store(cm.ctx).NodeGroups(cm.cluster.Name).Delete(ng.Name)
			}
		}
	}
	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Master Instance",
		Message:  fmt.Sprintf("Found master instance with name %v", cm.cluster.Spec.KubernetesMasterName),
	})
	if !dryRun {
		if err = cm.conn.deleteMaster(); err != nil {
			//return
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
				//return
			}
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Master persistant disk",
		Message:  fmt.Sprintf("Will delete master persistant with name %v", cm.namer.MasterPDName()),
	})

	if !dryRun {
		if err = cm.conn.deleteDisk(); err != nil {
			//return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Route",
		Message:  fmt.Sprintf("Route will be delete"),
	})
	if !dryRun {
		if err = cm.conn.deleteRoutes(); err != nil {
			//	return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "A Record",
		Message:  fmt.Sprintf("Found cluster apps A record %v, External domain %v and internal domain %v", Extra(cm.ctx).Domain(cm.cluster.Name), Extra(cm.ctx).ExternalDomain(cm.cluster.Name), Extra(cm.ctx).InternalDomain(cm.cluster.Name)),
	})
	if !dryRun {
		if err = DeleteARecords(cm.ctx, cm.cluster); err != nil {
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

	upm := NewUpgradeManager(cm.ctx, cm.conn, kc, cm.cluster)
	if !dryRun {
		var a []api.Action
		a, err = upm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a...)

		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}
	return
}
