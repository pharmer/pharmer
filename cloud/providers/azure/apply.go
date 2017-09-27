package azure

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	armstorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	azstore "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/TamalSaha/go-oneliners"
	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
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
	fmt.Println(clusterDelete)

	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster); err != nil {
		return
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return
	}
	acts = make([]api.Action, 0)

	if cm.cluster.Status.Phase == "" {
		err = fmt.Errorf("cluster `%s` is in unknown status", cm.cluster.Name)
		return
	}

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
	oneliners.FILE()
	// Common stuff
	if err = cm.conn.detectUbuntuImage(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	if found, _ := cm.getResourceGroup(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Resource group",
			Message:  "Resource group will be created",
		})
		oneliners.FILE()
		if rt != api.DryRun {
			if _, err = cm.ensureResourceGroup(); err != nil {
				fmt.Println(err, "********")
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			Logger(cm.ctx).Infof("Resource group %v in zone %v created", cm.namer.ResourceGroupName(), cm.cluster.Spec.Cloud.Zone)
		}
	} else {
		if clusterDelete {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Resource group",
				Message:  "Resource group will be deleted",
			})
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "Resource group",
				Message:  "Resource group found",
			})
		}
	}

	var as compute.AvailabilitySet
	if as, err = cm.getAvailablitySet(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Availablity set",
			Message:  fmt.Sprintf("Availablity set %v created", cm.namer.AvailablitySetName()),
		})
		if rt != api.DryRun {
			if as, err = cm.ensureAvailablitySet(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return
				Logger(cm.ctx).Infof("Availablity set %v created", cm.namer.AvailablitySetName())
			}
		}
	}

	var sa armstorage.Account
	if sa, err = cm.getStorageAccount(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Storage account",
			Message:  fmt.Sprintf("Storage account will be created"),
		})
		if rt != api.DryRun {
			if sa, err = cm.createStorageAccount(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				err = errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	}

	var vn network.VirtualNetwork
	if vn, err = cm.getVirtualNetwork(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Virtual network",
			Message:  fmt.Sprintf("Virtual network %v will be created", cm.namer.VirtualNetworkName()),
		})
		if rt != api.DryRun {
			if vn, err = cm.ensureVirtualNetwork(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	}

	var sg network.SecurityGroup
	if sg, err = cm.getNetworkSecurityGroup(); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Network security group",
			Message:  fmt.Sprintf("Network security group %v will be created", cm.namer.NetworkSecurityGroupName()),
		})
		if rt != api.DryRun {
			if sg, err = cm.createNetworkSecurityGroup(); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	}

	var sn network.Subnet
	if sn, err = cm.getSubnetID(&vn); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Subnet id",
			Message:  fmt.Sprintf("Subnet %v will be created", cm.namer.SubnetName()),
		})
		if rt != api.DryRun {
			if sn, err = cm.createSubnetID(&vn, &sg); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	nodeGroups, err := Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	var masterNG *api.NodeGroup
	var totalNodes int64 = 0
	for _, ng := range nodeGroups {
		if ng.IsMaster() {
			masterNG = ng
		} else {
			totalNodes += ng.Spec.Nodes
		}
	}
	fmt.Println(totalNodes)
	cm.cluster.Spec.MasterSKU = "Standard_D2_v2"

	var masterPIP network.PublicIPAddress

	if masterPIP, err = im.getPublicIP(cm.namer.PublicIPName(cm.namer.MasterName())); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Public ip address",
			Message:  fmt.Sprintf("Public ip will be created"),
		})
		if rt != api.DryRun {
			if masterPIP, err = im.createPublicIP(cm.namer.PublicIPName(cm.namer.MasterName()), network.Static); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			cm.cluster.Spec.MasterReservedIP = String(masterPIP.IPAddress)
		}
	} else {
		cm.cluster.Spec.MasterReservedIP = String(masterPIP.IPAddress)
	}

	// @dipta
	if cm.cluster.Spec.MasterExternalIP == "" {
		cm.cluster.Spec.MasterExternalIP = cm.cluster.Spec.MasterReservedIP
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
	if masterNIC, err = im.getNetworkInterface(cm.namer.NetworkInterfaceName(cm.cluster.Spec.KubernetesMasterName)); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Public ip address",
			Message:  fmt.Sprintf("Public ip will be created"),
		})
		if rt != api.DryRun {
			if masterNIC, err = im.createNetworkInterface(cm.namer.NetworkInterfaceName(cm.cluster.Spec.KubernetesMasterName), sg, sn, network.Static, cm.cluster.Spec.MasterInternalIP, masterPIP); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	}

	if found, _ := cm.getNetworkSecurityRule(); !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Network security rule",
			Message:  fmt.Sprintf("All network security will be created"),
		})
		if rt != api.DryRun {
			if err = cm.createNetworkSecurityRule(&sg); err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
		}
	}

	var masterVM compute.VirtualMachine
	if masterVM, err = im.getVirtualMachine(cm.namer.MasterName()); err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master virtual machine",
			Message:  fmt.Sprintf("Virtual machine %v will be created", cm.namer.MasterName()),
		})
		if rt != api.DryRun {
			var masterScript string
			masterScript, err = RenderStartupScript(cm.ctx, cm.cluster, api.RoleMaster, "")
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			masterVM, err = im.createVirtualMachine(masterNIC, as, sa, cm.namer.MasterName(), masterScript, cm.cluster.Spec.MasterSKU)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			var masterInstance *api.Node
			masterInstance, err = im.newKubeInstance(masterVM, masterNIC, masterPIP)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			masterInstance.Spec.Role = api.RoleMaster
			cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
			cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP

			fmt.Println(cm.cluster.Spec.MasterExternalIP, "------------------------------->")
			Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

			err = EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
			if err != nil {
				errors.FromErr(err).WithContext(cm.ctx).Err()
				return
			}
			masterNG.Status.Nodes = int32(1)
			Store(cm.ctx).NodeGroups(cm.cluster.Name).Update(masterNG)
			Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)
			// needed to get master_internal_ip
			if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
				cm.cluster.Status.Reason = err.Error()
				return acts, err
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Found master instance with name %v", cm.cluster.Spec.KubernetesMasterName),
		})
		masterInstance, _ := im.newKubeInstance(masterVM, masterNIC, masterPIP)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			errors.FromErr(err).WithContext(cm.ctx).Err()
			return
		}
		masterInstance.Spec.Role = api.RoleMaster
		cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
		cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
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
			im: im,
		}
		if clusterDelete || node.DeletionTimestamp != nil {
			instanceGroupName := igm.cm.namer.GetNodeGroupName(igm.instance.Type.Sku)
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Node Group",
				Message:  fmt.Sprintf("Node group %v  will be deleted", instanceGroupName),
			})
			if rt != api.DryRun {
				//err = igm.deleteNodeGroup(igm.instance.Type.Sku)
				Store(cm.ctx).NodeGroups(cm.cluster.Name).Delete(node.Name)
			}
		} else {
			/*act, _ :=*/ igm.AdjustNodeGroup(rt)
			//acts = append(acts, act...)
			if rt != api.DryRun {
				node.Status.Nodes = (int32)(node.Spec.Nodes)
				Store(cm.ctx).NodeGroups(cm.cluster.Name).UpdateStatus(node)
			}
		}

	}

	if rt != api.DryRun {
		time.Sleep(1 * time.Minute)

		for _, ng := range nodeGroups {
			if ng.IsMaster() {
				continue
			}
			fmt.Println(ng.Spec.Template.Spec.SKU, "................")
			groupName := cm.namer.GetNodeGroupName(ng.Spec.Template.Spec.SKU)
			_, providerInstances, _ := im.GetNodeGroup(groupName)

			runningInstance := make(map[string]*api.Node)
			for _, node := range providerInstances {
				fmt.Println(node.Name, "************<<")
				runningInstance[node.Name] = node
			}

			clusterInstance, _ := GetClusterIstance(cm.ctx, cm.cluster, groupName)
			for _, node := range clusterInstance {
				fmt.Println(node, "----------->>>")
				if _, found := runningInstance[node]; !found {
					err = DeleteClusterInstance(cm.ctx, cm.cluster, node)
					fmt.Println(err)
				}
			}
		}

		if !clusterDelete {
			cm.cluster.Status.Phase = api.ClusterReady
		} else {
			cm.cluster.Status.Phase = api.ClusterDeleted
		}
		Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		Store(cm.ctx).Clusters().Update(cm.cluster)
	}

	//for _, ng := range req.NodeGroups {
	//	Logger(cm.ctx).Infof("Creating %v node with sku %v", ng.Count, ng.Sku)
	//	igm := &NodeGroupManager{
	//		cm: cm,
	//		instance: Instance{
	//			Type: InstanceType{
	//				ContextVersion: cm.cluster.Generation,
	//				Sku:            ng.Sku,
	//
	//				Master:       false,
	//				SpotInstance: false,
	//			},
	//			Stats: GroupStats{
	//				Count: ng.Count,
	//			},
	//		},
	//		im: im,
	//	}
	//	err = igm.AdjustNodeGroup()
	//}

	/*
		Logger(cm.ctx).Info("Waiting for cluster initialization")

		// Wait for master A record to propagate
		if err := EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		// wait for nodes to start
		if err := WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	*/

	cm.cluster.Status.Phase = api.ClusterReady
	return
}

// IP >>>>>>>>>>>>>>>>
// TODO(tamal): if cluster.Spec.ctx.MasterReservedIP == "auto"
//	name := cluster.Spec.ctx.KubernetesMasterName + "-pip"
//	// cluster.Spec.ctx.MasterExternalIP = *ip.IPAddress
//	cluster.Spec.ctx.MasterReservedIP = *ip.IPAddress
//	// cluster.Spec.ctx.ApiServerUrl = "https://" + *ip.IPAddress

func (cm *ClusterManager) getResourceGroup() (bool, error) {
	_, err := cm.conn.groupsClient.Get(cm.namer.ResourceGroupName())
	return err == nil, err
}

func (cm *ClusterManager) ensureResourceGroup() (resources.Group, error) {
	req := resources.Group{
		Name:     StringP(cm.namer.ResourceGroupName()),
		Location: StringP(cm.cluster.Spec.Cloud.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(cm.cluster.Name),
		},
	}
	return cm.conn.groupsClient.CreateOrUpdate(cm.namer.ResourceGroupName(), req)
}

func (cm *ClusterManager) getAvailablitySet() (compute.AvailabilitySet, error) {
	return cm.conn.availabilitySetsClient.Get(cm.namer.ResourceGroupName(), cm.namer.AvailablitySetName())
}

func (cm *ClusterManager) ensureAvailablitySet() (compute.AvailabilitySet, error) {
	name := cm.namer.AvailablitySetName()
	req := compute.AvailabilitySet{
		Name:     StringP(name),
		Location: StringP(cm.cluster.Spec.Cloud.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(cm.cluster.Name),
		},
	}
	return cm.conn.availabilitySetsClient.CreateOrUpdate(cm.namer.ResourceGroupName(), name, req)
}

func (cm *ClusterManager) getVirtualNetwork() (network.VirtualNetwork, error) {
	return cm.conn.virtualNetworksClient.Get(cm.namer.ResourceGroupName(), cm.namer.VirtualNetworkName(), "")
}

func (cm *ClusterManager) ensureVirtualNetwork() (network.VirtualNetwork, error) {
	name := cm.namer.VirtualNetworkName()
	req := network.VirtualNetwork{
		Name:     StringP(name),
		Location: StringP(cm.cluster.Spec.Cloud.Zone),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{cm.cluster.Spec.Networking.NonMasqueradeCIDR},
			},
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(cm.cluster.Name),
		},
	}

	_, errchan := cm.conn.virtualNetworksClient.CreateOrUpdate(cm.namer.ResourceGroupName(), name, req, nil)
	err := <-errchan
	if err != nil {
		return network.VirtualNetwork{}, err
	}
	Logger(cm.ctx).Infof("Virtual network %v created", name)
	return cm.conn.virtualNetworksClient.Get(cm.namer.ResourceGroupName(), name, "")
}

func (cm *ClusterManager) getNetworkSecurityGroup() (network.SecurityGroup, error) {
	return cm.conn.securityGroupsClient.Get(cm.namer.ResourceGroupName(), cm.namer.NetworkSecurityGroupName(), "")
}

func (cm *ClusterManager) createNetworkSecurityGroup() (network.SecurityGroup, error) {
	securityGroupName := cm.namer.NetworkSecurityGroupName()
	securityGroup := network.SecurityGroup{
		Name:     StringP(securityGroupName),
		Location: StringP(cm.cluster.Spec.Cloud.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(cm.cluster.Name),
		},
	}
	_, errchan := cm.conn.securityGroupsClient.CreateOrUpdate(cm.namer.ResourceGroupName(), securityGroupName, securityGroup, nil)
	err := <-errchan
	if err != nil {
		return network.SecurityGroup{}, err
	}
	Logger(cm.ctx).Infof("Network security group %v created", securityGroupName)
	return cm.conn.securityGroupsClient.Get(cm.namer.ResourceGroupName(), securityGroupName, "")
}

func (cm *ClusterManager) getSubnetID(vn *network.VirtualNetwork) (network.Subnet, error) {
	return cm.conn.subnetsClient.Get(cm.namer.ResourceGroupName(), *vn.Name, cm.namer.SubnetName(), "")
}

func (cm *ClusterManager) createSubnetID(vn *network.VirtualNetwork, sg *network.SecurityGroup) (network.Subnet, error) {
	name := cm.namer.SubnetName()
	var routeTable network.RouteTable
	var err error
	if routeTable, err = cm.getRouteTable(); err != nil {
		if routeTable, err = cm.createRouteTable(); err != nil {
			return network.Subnet{}, err
		}
	}
	req := network.Subnet{
		Name: StringP(name),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: sg.ID,
			},
			AddressPrefix: StringP(cm.cluster.Spec.Cloud.Azure.SubnetCIDR),
			RouteTable: &network.RouteTable{
				ID: routeTable.ID,
			},
		},
	}

	_, errchan := cm.conn.subnetsClient.CreateOrUpdate(cm.namer.ResourceGroupName(), *vn.Name, name, req, nil)
	err = <-errchan
	if err != nil {
		return network.Subnet{}, err
	}
	Logger(cm.ctx).Infof("Subnet name %v created", name)
	return cm.conn.subnetsClient.Get(cm.namer.ResourceGroupName(), *vn.Name, name, "")
}

func (cm *ClusterManager) getRouteTable() (network.RouteTable, error) {
	return cm.conn.routeTablesClient.Get(cm.namer.ResourceGroupName(), cm.namer.RouteTableName(), "")
}

func (cm *ClusterManager) createRouteTable() (network.RouteTable, error) {
	name := cm.namer.RouteTableName()
	req := network.RouteTable{
		Name:     StringP(name),
		Location: StringP(cm.cluster.Spec.Cloud.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(cm.cluster.Name),
		},
	}
	_, errchan := cm.conn.routeTablesClient.CreateOrUpdate(cm.namer.ResourceGroupName(), name, req, nil)
	err := <-errchan
	if err != nil {
		return network.RouteTable{}, err
	}
	Logger(cm.ctx).Infof("Route table %v created", name)
	return cm.conn.routeTablesClient.Get(cm.namer.ResourceGroupName(), name, "")
}

func (cm *ClusterManager) getNetworkSecurityRule() (bool, error) {
	_, err := cm.conn.securityRulesClient.Get(cm.namer.ResourceGroupName(), cm.namer.NetworkSecurityGroupName(), cm.namer.NetworkSecurityRule("ssh"))
	if err != nil {
		return false, err
	}
	_, err = cm.conn.securityRulesClient.Get(cm.namer.ResourceGroupName(), cm.namer.NetworkSecurityGroupName(), cm.namer.NetworkSecurityRule("ssl"))
	if err != nil {
		return false, err
	}
	_, err = cm.conn.securityRulesClient.Get(cm.namer.ResourceGroupName(), cm.namer.NetworkSecurityGroupName(), cm.namer.NetworkSecurityRule("masterssl"))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (cm *ClusterManager) createNetworkSecurityRule(sg *network.SecurityGroup) error {
	sshRuleName := cm.namer.NetworkSecurityRule("ssh")
	sshRule := network.SecurityRule{
		Name: StringP(sshRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access: network.SecurityRuleAccessAllow,
			DestinationAddressPrefix: StringP("*"),
			DestinationPortRange:     StringP("22"),
			Direction:                network.SecurityRuleDirectionInbound,
			Priority:                 Int32P(100),
			Protocol:                 network.SecurityRuleProtocolTCP,
			SourceAddressPrefix:      StringP("*"),
			SourcePortRange:          StringP("*"),
		},
	}
	_, errchan := cm.conn.securityRulesClient.CreateOrUpdate(cm.namer.ResourceGroupName(), *sg.Name, sshRuleName, sshRule, nil)
	err := <-errchan
	if err != nil {
		return err
	}
	Logger(cm.ctx).Infof("Network security rule %v created", sshRuleName)
	sslRuleName := cm.namer.NetworkSecurityRule("ssl")
	sslRule := network.SecurityRule{
		Name: StringP(sshRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access: network.SecurityRuleAccessAllow,
			DestinationAddressPrefix: StringP("*"),
			DestinationPortRange:     StringP("443"),
			Direction:                network.SecurityRuleDirectionInbound,
			Priority:                 Int32P(110),
			Protocol:                 network.SecurityRuleProtocolTCP,
			SourceAddressPrefix:      StringP("*"),
			SourcePortRange:          StringP("*"),
		},
	}
	_, errchan = cm.conn.securityRulesClient.CreateOrUpdate(cm.namer.ResourceGroupName(), *sg.Name, sslRuleName, sslRule, nil)
	err = <-errchan
	if err != nil {
		return err
	}
	Logger(cm.ctx).Infof("Network security rule %v created", sslRuleName)

	mastersslRuleName := cm.namer.NetworkSecurityRule("masterssl")
	mastersslRule := network.SecurityRule{
		Name: StringP(mastersslRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access: network.SecurityRuleAccessAllow,
			DestinationAddressPrefix: StringP("*"),
			DestinationPortRange:     StringP("6443"),
			Direction:                network.SecurityRuleDirectionInbound,
			Priority:                 Int32P(120),
			Protocol:                 network.SecurityRuleProtocolTCP,
			SourceAddressPrefix:      StringP("*"),
			SourcePortRange:          StringP("*"),
		},
	}
	_, errchan = cm.conn.securityRulesClient.CreateOrUpdate(cm.namer.ResourceGroupName(), *sg.Name, mastersslRuleName, mastersslRule, nil)
	err = <-errchan
	if err != nil {
		return err
	}
	Logger(cm.ctx).Infof("Network security rule %v created", mastersslRuleName)

	return err
}

func (cm *ClusterManager) getStorageAccount() (armstorage.Account, error) {
	storageName := cm.cluster.Spec.Cloud.Azure.CloudConfig.StorageAccountName
	return cm.conn.storageClient.GetProperties(cm.namer.ResourceGroupName(), storageName)
}

func (cm *ClusterManager) createStorageAccount() (armstorage.Account, error) {
	storageName := cm.cluster.Spec.Cloud.Azure.CloudConfig.StorageAccountName
	req := armstorage.AccountCreateParameters{
		Location: StringP(cm.cluster.Spec.Cloud.Zone),
		Sku: &armstorage.Sku{
			Name: armstorage.StandardLRS,
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(cm.cluster.Name),
		},
	}
	_, errchan := cm.conn.storageClient.Create(cm.namer.ResourceGroupName(), storageName, req, nil)
	err := <-errchan
	if err != nil {
		return armstorage.Account{}, err
	}
	Logger(cm.ctx).Infof("Storage account %v created", storageName)
	keys, err := cm.conn.storageClient.ListKeys(cm.namer.ResourceGroupName(), storageName)
	if err != nil {
		return armstorage.Account{}, err
	}
	storageClient, err := azstore.NewBasicClient(storageName, *(*(keys.Keys))[0].Value)
	if err != nil {
		return armstorage.Account{}, err
	}

	bs := storageClient.GetBlobService()
	_, err = bs.GetContainerReference(cm.namer.StorageContainerName()).CreateIfNotExists(&azstore.CreateContainerOptions{Access: azstore.ContainerAccessTypeContainer})
	if err != nil {
		return armstorage.Account{}, err
	}
	return cm.conn.storageClient.GetProperties(cm.namer.ResourceGroupName(), storageName)
}

/*
for i := int64(0); i < ng.Count; i++ {
			nodeName := cm.namer.GenNodeName(ng.Sku)

			nodePIP, err := im.createPublicIP(cm.namer.PublicIPName(nodeName), network.Dynamic)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			nodeNIC, err := im.createNetworkInterface(cm.namer.NetworkInterfaceName(nodeName), sn, network.Dynamic, "", nodePIP)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			nodeScript := im.RenderStartupScript(cm.ctx.NewScriptOptions(), ng.Sku, api.RoleKubernetesPool)
			nodeVM, err := im.createVirtualMachine(nodeNIC, as, sa, nodeName, nodeScript, ng.Sku)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			nodePIP, err = im.getPublicIP(cm.namer.PublicIPName(nodeName))
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			ki, err := im.newKubeInstance(nodeVM, nodeNIC, nodePIP)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			ki.Role = api.RoleKubernetesPool
			cm.ins.Instances = append(cm.ins.Instances, ki)
			// cm.ins.Instances = append(cm.ins.Instances, ki)
		}
*/
