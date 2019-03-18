package azure

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	armstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	capiAzure "github.com/pharmer/pharmer/apis/v1beta1/azure"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	cm.conn.namer = cm.namer

	// Common stuff
	/*	if err = cm.conn.detectUbuntuImage(); err != nil {
		return nil, errors.Wrap(err, ID(cm.ctx))
	}*/

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
			if _, err := Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
				return nil, err
			}
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
		nodeGroups, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		replica := int32(0)
		for _, ng := range nodeGroups {
			ng.Spec.Replicas = &replica
			_, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Update(ng)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		if err := cm.applyScale(dryRun); err != nil {
			return nil, err
		}
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
		Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
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

func (cm *ClusterManager) applyCreate(dryRun bool) ([]api.Action, error) {
	var acts []api.Action

	if err := cm.SetupCerts(); err != nil {
		return nil, err
	}

	found, err := cm.conn.getResourceGroup()
	if err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Resource group",
			Message:  "Resource group will be created",
		})
		if !dryRun {
			if _, err = cm.conn.ensureResourceGroup(); err != nil {
				return acts, err
			}
			Logger(cm.ctx).Infof("Resource group %v in zone %v created", cm.namer.ResourceGroupName(), cm.cluster.Spec.Config.Cloud.Zone)
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Resource group",
			Message:  "Resource group found",
		})
	}

	var as compute.AvailabilitySet
	if as, err = cm.conn.getAvailabilitySet(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Availability set",
			Message:  fmt.Sprintf("Availability set %v will be created", cm.namer.AvailabilitySetName()),
		})
		if !dryRun {
			if as, err = cm.conn.ensureAvailabilitySet(); err != nil {
				return acts, err
			}
			Logger(cm.ctx).Infof("Availability set %v created", cm.namer.AvailabilitySetName())
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Availability set",
			Message:  fmt.Sprintf("Availability set %v found", cm.namer.AvailabilitySetName()),
		})
	}
	var sa armstorage.Account
	if sa, err = cm.conn.getStorageAccount(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Storage account",
			Message:  fmt.Sprintf("Storage account %v will be created", cm.cluster.Spec.Config.Cloud.Azure.StorageAccountName),
		})
		if !dryRun {
			if sa, err = cm.conn.createStorageAccount(); err != nil {
				return acts, err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Storage account",
			Message:  fmt.Sprintf("Storage account %v found", cm.cluster.Spec.Config.Cloud.Azure.StorageAccountName),
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
				return acts, err
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
				return acts, err
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
				return acts, err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Subnet id",
			Message:  fmt.Sprintf("Subnet %v found", cm.namer.SubnetName()),
		})
	}

	var lbPIP network.PublicIPAddress
	if lbPIP, err = cm.conn.getPublicIP(cm.namer.LoadBalancerName()); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Balancer public ip address",
			Message:  fmt.Sprintf("Load Balancer public ip will be created"),
		})
		if !dryRun {
			if lbPIP, err = cm.conn.createPublicIP(cm.namer.LoadBalancerName()); err != nil {
				cm.cluster.Status.Reason = err.Error()
				//errors.Wrap(err, ID(cm.ctx))
				return acts, err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Load Balancer ip address",
			Message:  fmt.Sprintf("Load Balancer ip found"),
		})
	}

	var lb network.LoadBalancer
	if lb, err = cm.conn.findLoadBalancer(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Balancer",
			Message:  fmt.Sprintf("Load Balancer %v will be created", cm.namer.LoadBalancerName()),
		})
		if !dryRun {
			if lb, err = cm.conn.createLoadBalancer(&lbPIP); err != nil {
				return acts, err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Load Balancer",
			Message:  fmt.Sprintf("Load Balancer %v found", cm.namer.SubnetName()),
		})
	}
	cm.cluster.Status.Cloud.Azure.LBDNS = *lbPIP.DNSSettings.Fqdn

	var masterPIP network.PublicIPAddress
	if masterPIP, err = cm.conn.getPublicIP(cm.namer.MasterName()); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master public ip address",
			Message:  fmt.Sprintf("Master public ip will be created"),
		})
		if !dryRun {
			if masterPIP, err = cm.conn.createPublicIP(cm.namer.MasterName()); err != nil {
				cm.cluster.Status.Reason = err.Error()
				//errors.Wrap(err, ID(cm.ctx))
				return acts, err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master ip address",
			Message:  fmt.Sprintf("Master ip found"),
		})
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
			if masterNIC, err = cm.conn.createNetworkInterface(cm.namer.NetworkInterfaceName(cm.namer.MasterName()), sg, sn, lb, &masterPIP); err != nil {
				return acts, err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master network interface",
			Message:  fmt.Sprintf("Masater network interface %v found", cm.namer.NetworkInterfaceName(cm.namer.MasterName())),
		})
	}

	var machines []*clusterapi.Machine
	machines, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return acts, err
	}

	var masterMachine *clusterapi.Machine
	masterMachine, err = api.GetMasterMachine(machines)
	if err != nil {
		return acts, err
	}

	//var masterVM compute.VirtualMachine
	if _, err := cm.conn.getVirtualMachine(cm.namer.MasterName()); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master virtual machine",
			Message:  fmt.Sprintf("Virtual machine %v will be created", cm.namer.MasterName()),
		})
		if !dryRun {
			script, err := cm.conn.renderStartupScript(cm.cluster, masterMachine, cm.owner, "")
			if err != nil {
				return acts, err
			}

			vm, err := cm.conn.createVirtualMachine(masterNIC, as, sa, cm.namer.MasterName(), script, masterMachine)
			if err != nil {
				return acts, err
			}
			//var masterInstance *api.NodeInfo
			//if masterInstance, err = cm.conn.newKubeInstance(masterVM, masterNIC, lbPIP); err != nil {
			//	return
			//}
			//
			//oneliners.PrettyJson(masterInstance)

			nodeAddresses := []core.NodeAddress{
				{
					Type:    core.NodeExternalDNS,
					Address: cm.cluster.Status.Cloud.Azure.LBDNS,
				},
			}

			if err = cm.cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
				return nil, err
			}

			if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
				return nil, err
			}

			// update master machine status
			statusConfig := capiAzure.AzureMachineProviderStatus{
				VMID: vm.ID,
			}

			rawStatus, err := capiAzure.EncodeMachineStatus(&statusConfig)
			if err != nil {
				return nil, err
			}
			masterMachine.Status.ProviderStatus = rawStatus

			// update in pharmer file
			masterMachine, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Update(masterMachine)
			if err != nil {
				return nil, errors.Wrap(err, "error updating master machine in pharmer storage")
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Found master instance with name %v", cm.namer.MasterName()),
		})
	}

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return acts, err
	}

	// wait for nodes to start
	if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
		return acts, err
	}

	ca, err := NewClusterApi(cm.ctx, cm.cluster, cm.owner, "cloud-provider-system", kc, cm.conn)
	if err != nil {
		return acts, err
	}

	if err := ca.Apply(ClusterAPIComponents); err != nil {
		return acts, err
	}

	cm.cluster.Status.Phase = api.ClusterReady
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
		return acts, err
	}

	return acts, err
}

func (cm *ClusterManager) applyScale(dryRun bool) error {
	Logger(cm.ctx).Infoln("scaling machine set")

	//var msc *clusterv1.MachineSet
	machineSets, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	bc, err := GetBooststrapClient(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return err
	}
	var data []byte
	for _, machineSet := range machineSets {
		if machineSet.DeletionTimestamp != nil {
			machineSet.DeletionTimestamp = nil
			if data, err = json.Marshal(machineSet); err != nil {
				return err
			}

			if err = bc.Delete(string(data)); err != nil {
				return nil
			}
			if err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Delete(machineSet.Name); err != nil {
				return nil
			}
		}

		existingMachineSet, err := bc.GetMachineSetObjects()
		if err != nil {
			return err
		}

		if data, err = json.Marshal(machineSet); err != nil {
			return err
		}
		found := false
		for _, ems := range existingMachineSet {
			if ems.Name == machineSet.Name {
				found = true
				if err = bc.Apply(string(data)); err != nil {
					return err
				}
				break
			}
		}

		if !found {
			if err = bc.CreateMachineSetObjects([]*clusterapi.MachineSet{machineSet}, bc.GetContextNamespace()); err != nil {
				return err
			}
		}
	}

	return nil
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
		// Failed
		cm.cluster.Status.Phase = api.ClusterDeleted
		_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
		if err != nil {
			return
		}
	}

	return
}

func (cm *ClusterManager) applyUpgrade(dryRun bool) (acts []api.Action, err error) {
	/*var kc kubernetes.Interface
	if kc, err = cm.GetAdminClient(); err != nil {
		return
	}

	upm := NewUpgradeManager(cm.ctx, cm, kc, cm.cluster, cm.owner)
	a, err := upm.Apply(dryRun)
	if err != nil {
		return
	}
	acts = append(acts, a...)
	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}*/
	return
}
