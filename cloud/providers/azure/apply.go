package azure

import (
	"bytes"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	armstorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	core "k8s.io/api/core/v1"
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

	// Common stuff
	if err = cm.conn.detectUbuntuImage(); err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if cm.cluster.Status.Phase == api.ClusterUpgrading {
		return nil, fmt.Errorf("cluster `%s` is upgrading. Retry after cluster returns to Ready state", cm.cluster.Name)
	}
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

	/*defer func(releaseReservedIp bool) {
		if cm.cluster.Status.Phase == api.ClusterPending {
			cm.cluster.Status.Phase = api.ClusterFailing
		}
		Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterReady {
			Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.Delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")*/
}

// IP >>>>>>>>>>>>>>>>
// TODO(tamal): if cluster.Spec.ctx.MasterReservedIP == "auto"
//	name := cluster.Spec.ctx.KubernetesMasterName + "-pip"
//	// cluster.Spec.ctx.MasterExternalIP = *ip.IPAddress
//	cluster.Spec.ctx.MasterReservedIP = *ip.IPAddress
//	// cluster.Spec.ctx.ApiServerUrl = "https://" + *ip.IPAddress

func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	var found bool
	found, _ = cm.conn.getResourceGroup()
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Resource group",
			Message:  "Resource group will be created",
		})
		if !dryRun {
			if _, err = cm.conn.ensureResourceGroup(); err != nil {
				return
			}
			Logger(cm.ctx).Infof("Resource group %v in zone %v created", cm.namer.ResourceGroupName(), cm.cluster.Spec.Cloud.Zone)
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Resource group",
			Message:  "Resource group found",
		})
	}
	var as compute.AvailabilitySet
	if as, err = cm.conn.getAvailablitySet(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Availablity set",
			Message:  fmt.Sprintf("Availablity set %v will be created", cm.namer.AvailablitySetName()),
		})
		if !dryRun {
			if as, err = cm.conn.ensureAvailablitySet(); err != nil {
				return
			}
			Logger(cm.ctx).Infof("Availablity set %v created", cm.namer.AvailablitySetName())
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Availablity set",
			Message:  fmt.Sprintf("Availablity set %v found", cm.namer.AvailablitySetName()),
		})
	}
	var sa armstorage.Account
	if sa, err = cm.conn.getStorageAccount(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Storage account",
			Message:  fmt.Sprintf("Storage account %v will be created", cm.cluster.Status.Cloud.Azure.CloudConfig.StorageAccountName),
		})
		if !dryRun {
			if sa, err = cm.conn.createStorageAccount(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Storage account",
			Message:  fmt.Sprintf("Storage account %v found", cm.cluster.Status.Cloud.Azure.CloudConfig.StorageAccountName),
		})
	}

	var vn network.VirtualNetwork
	if vn, err = cm.conn.getVirtualNetwork(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Virtual network",
			Message:  fmt.Sprintf("Virtual network %v will be created", cm.namer.VirtualNetworkName()),
		})
		if !dryRun {
			if vn, err = cm.conn.ensureVirtualNetwork(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Virtual network",
			Message:  fmt.Sprintf("Virtual network %v found", cm.namer.VirtualNetworkName()),
		})
	}

	var sg network.SecurityGroup
	if sg, err = cm.conn.getNetworkSecurityGroup(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Network security group",
			Message:  fmt.Sprintf("Network security group %v will be created", cm.namer.NetworkSecurityGroupName()),
		})
		if !dryRun {
			if sg, err = cm.conn.createNetworkSecurityGroup(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Network security group",
			Message:  fmt.Sprintf("Network security group %v found", cm.namer.NetworkSecurityGroupName()),
		})
	}

	var sn network.Subnet
	if sn, err = cm.conn.getSubnetID(&vn); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Subnet id",
			Message:  fmt.Sprintf("Subnet %v will be created", cm.namer.SubnetName()),
		})
		if !dryRun {
			if sn, err = cm.conn.createSubnetID(&vn, &sg); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Subnet id",
			Message:  fmt.Sprintf("Subnet %v found", cm.namer.SubnetName()),
		})
	}

	nodeGroups, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	var masterNG *api.NodeGroup
	masterNG = FindMasterNodeGroup(nodeGroups)
	if masterNG.Spec.Template.Spec.SKU == "" {
		masterNG.Spec.Template.Spec.SKU = "Standard_D2_v2"
		masterNG, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).Update(masterNG)
		if err != nil {
			return
		}
	}

	//Creating Master
	var masterPIP network.PublicIPAddress

	if masterPIP, err = cm.conn.getPublicIP(cm.namer.PublicIPName(cm.namer.MasterName())); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master public ip address",
			Message:  fmt.Sprintf("Master public ip will be created"),
		})
		if !dryRun {
			if masterPIP, err = cm.conn.createPublicIP(cm.namer.PublicIPName(cm.namer.MasterName()), network.Static); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			cm.cluster.Status.ReservedIPs = append(cm.cluster.Status.ReservedIPs, api.ReservedIP{
				IP: String(masterPIP.IPAddress),
			})
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master public ip address",
			Message:  fmt.Sprintf("Master Public ip found"),
		})
	}

	// IP >>>>>>>>>>>>>>>>
	// TODO(tamal): if cluster.Spec.ctx.MasterReservedIP == "auto"
	//	name := cluster.Spec.ctx.KubernetesMasterName + "-pip"
	//	// cluster.Spec.ctx.MasterExternalIP = *ip.IPAddress
	//	cluster.Spec.ctx.MasterReservedIP = *ip.IPAddress
	//	// cluster.Spec.ctx.ApiServerUrl = "https://" + *ip.IPAddress

	// needed for master start-up config
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	// Master Stuff
	var masterNIC network.Interface
	if masterNIC, err = cm.conn.getNetworkInterface(cm.namer.NetworkInterfaceName(cm.namer.MasterName())); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master network interface",
			Message:  fmt.Sprintf("Masater network interface %v will be created", cm.namer.NetworkInterfaceName(cm.namer.MasterName())),
		})
		if !dryRun {
			if masterNIC, err = cm.conn.createNetworkInterface(cm.namer.NetworkInterfaceName(cm.namer.MasterName()), sg, sn, network.Static, cm.cluster.Spec.MasterInternalIP, masterPIP); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master network interface",
			Message:  fmt.Sprintf("Masater network interface %v found", cm.namer.NetworkInterfaceName(cm.namer.MasterName())),
		})
	}

	if found, _ := cm.conn.getNetworkSecurityRule(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Network security rule",
			Message:  fmt.Sprintf("All network security will be created"),
		})
		if !dryRun {
			if err = cm.conn.createNetworkSecurityRule(&sg); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Network security rule",
			Message:  fmt.Sprintf("All network security found"),
		})
	}

	var masterVM compute.VirtualMachine
	if masterVM, err = cm.conn.getVirtualMachine(cm.namer.MasterName()); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master virtual machine",
			Message:  fmt.Sprintf("Virtual machine %v will be created", cm.namer.MasterName()),
		})
		if !dryRun {
			var script bytes.Buffer
			if err = StartupScriptTemplate.ExecuteTemplate(&script, api.RoleMaster, newMasterTemplateData(cm.ctx, cm.cluster, masterNG)); err != nil {
				return
			}

			fmt.Println("----------------------------")
			fmt.Println(script.String())
			fmt.Println("----------------------------")
			masterVM, err = cm.conn.createVirtualMachine(masterNIC, as, sa, cm.namer.MasterName(), script.String(), masterNG.Spec.Template.Spec.SKU)
			if err != nil {
				return
			}
			var masterInstance *api.NodeInfo
			if masterInstance, err = cm.conn.newKubeInstance(masterVM, masterNIC, masterPIP); err != nil {
				return
			}
			cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
				Type:    core.NodeExternalIP,
				Address: masterInstance.PublicIP,
			})
			cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
				Type:    core.NodeInternalIP,
				Address: masterInstance.PrivateIP,
			})

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

			err = EnsureARecord(cm.ctx, cm.cluster, masterInstance.PublicIP, masterInstance.PrivateIP) // works for reserved or non-reserved mode
			if err != nil {
				return
			}

			cm.cluster.Status.Phase = api.ClusterReady
			if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Found master instance with name %v", cm.namer.MasterName()),
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
	var token string
	var kc kubernetes.Interface
	if cm.cluster.Status.Phase != api.ClusterPending {
		kc, err = cm.GetAdminClient()
		if err != nil {
			return
		}
		if !dryRun {
			if token, err = GetExistingKubeadmToken(kc); err != nil {
				return
			}
			if cm.cluster, err = Store(cm.ctx).Clusters().Update(cm.cluster); err != nil {
				return
			}
		}

	}
	for _, ng := range nodeGroups {
		if ng.IsMaster() {
			continue
		}
		igm := NewNodeGroupManager(cm.ctx, ng, cm.conn, kc, cm.cluster, token, nil, nil)
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
	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Resource group",
		Message:  "Resource group will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteResourceGroup(); err != nil {
			return
		}
		if err = DeleteARecords(cm.ctx, cm.cluster); err != nil {
			return
		}
		// Failed
		cm.cluster.Status.Phase = api.ClusterDeleted
		_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		if err != nil {
			return
		}
	}

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
