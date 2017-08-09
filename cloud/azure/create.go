package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	armstorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	azstore "github.com/Azure/azure-sdk-for-go/storage"
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/common"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = common.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn, err = NewConnector(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Save()

	defer func(releaseReservedIp bool) {
		if cm.ctx.Status == storage.KubernetesStatus_Pending {
			cm.ctx.Status = storage.KubernetesStatus_Failing
		}
		cm.ctx.Save()
		cm.ins.Save()
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Cluster %v is %v", cm.ctx.Name, cm.ctx.Status))
		if cm.ctx.Status != storage.KubernetesStatus_Ready {
			cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Cluster %v is deleting", cm.ctx.Name))
			cm.delete(&proto.ClusterDeleteRequest{
				Name:              cm.ctx.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.ctx.MasterReservedIP == "auto")

	// Common stuff
	_, err = cm.ensureResourceGroup()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Resource group %v in zone %v created", cm.namer.ResourceGroupName(), cm.ctx.Zone))
	as, err := cm.ensureAvailablitySet()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Availablity set %v created", cm.namer.AvailablitySetName()))
	sa, err := cm.createStorageAccount()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	vn, err := cm.ensureVirtualNetwork()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	sg, err := cm.createNetworkSecurityGroup()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	sn, err := cm.createSubnetID(&vn, &sg)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}

	masterPIP, err := im.createPublicIP(cm.namer.PublicIPName(cm.namer.MasterName()), network.Static)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.MasterReservedIP = types.String(masterPIP.IPAddress)
	cm.ctx.DetectApiServerURL()
	// IP >>>>>>>>>>>>>>>>
	// TODO(tamal): if cluster.ctx.MasterReservedIP == "auto"
	//	name := cluster.ctx.KubernetesMasterName + "-pip"
	//	// cluster.ctx.MasterExternalIP = *ip.IPAddress
	//	cluster.ctx.MasterReservedIP = *ip.IPAddress
	//	// cluster.ctx.ApiServerUrl = "https://" + *ip.IPAddress

	err = common.GenClusterCerts(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// needed for master start-up config
	if err = cm.ctx.Save(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.UploadStartupConfig()

	// Master Stuff
	masterNIC, err := im.createNetworkInterface(cm.namer.NetworkInterfaceName(cm.ctx.KubernetesMasterName), sn, network.Static, cm.ctx.MasterInternalIP, masterPIP)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.createNetworkSecurityRule(&sg)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterScript := im.RenderStartupScript(cm.ctx.NewScriptOptions(), cm.ctx.MasterSKU, system.RoleKubernetesMaster)
	masterVM, err := im.createVirtualMachine(masterNIC, as, sa, cm.namer.MasterName(), masterScript, cm.ctx.MasterSKU)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	ki, err := im.newKubeInstance(masterVM, masterNIC, masterPIP)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	ki.Role = system.RoleKubernetesMaster

	fmt.Println(cm.ctx.MasterExternalIP, "------------------------------->")
	cm.ins.Instances = append(cm.ins.Instances, ki)

	err = common.EnsureARecord(cm.ctx, ki) // works for reserved or non-reserved mode
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	fmt.Println(err, "<------------------------------->")

	for _, ng := range req.NodeGroups {
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Creating %v node with sku %v", ng.Count, ng.Sku))
		igm := &InstanceGroupManager{
			cm: cm,
			instance: common.Instance{
				Type: common.InstanceType{
					ContextVersion: cm.ctx.ContextVersion,
					Sku:            ng.Sku,

					Master:       false,
					SpotInstance: false,
				},
				Stats: common.GroupStats{
					Count: ng.Count,
				},
			},
			im: im,
		}
		err = igm.AdjustInstanceGroup()

	}

	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := common.EnsureDnsIPLookup(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// wait for nodes to start
	if err := common.ProbeKubeAPI(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = common.CheckComponentStatuses(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = common.WaitForReadyNodes(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Status = storage.KubernetesStatus_Ready
	return nil
}

// IP >>>>>>>>>>>>>>>>
// TODO(tamal): if cluster.ctx.MasterReservedIP == "auto"
//	name := cluster.ctx.KubernetesMasterName + "-pip"
//	// cluster.ctx.MasterExternalIP = *ip.IPAddress
//	cluster.ctx.MasterReservedIP = *ip.IPAddress
//	// cluster.ctx.ApiServerUrl = "https://" + *ip.IPAddress

func (cm *clusterManager) ensureResourceGroup() (resources.ResourceGroup, error) {
	req := resources.ResourceGroup{
		Name:     types.StringP(cm.namer.ResourceGroupName()),
		Location: types.StringP(cm.ctx.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(cm.ctx.Name),
		},
	}
	return cm.conn.groupsClient.CreateOrUpdate(cm.namer.ResourceGroupName(), req)
}

func (cm *clusterManager) ensureAvailablitySet() (compute.AvailabilitySet, error) {
	name := cm.namer.AvailablitySetName()
	req := compute.AvailabilitySet{
		Name:     types.StringP(name),
		Location: types.StringP(cm.ctx.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(cm.ctx.Name),
		},
	}
	return cm.conn.availabilitySetsClient.CreateOrUpdate(cm.namer.ResourceGroupName(), name, req)
}

func (cm *clusterManager) ensureVirtualNetwork() (network.VirtualNetwork, error) {
	name := cm.namer.VirtualNetworkName()
	req := network.VirtualNetwork{
		Name:     types.StringP(name),
		Location: types.StringP(cm.ctx.Zone),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{cm.ctx.NonMasqueradeCidr},
			},
		},
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(cm.ctx.Name),
		},
	}

	_, err := cm.conn.virtualNetworksClient.CreateOrUpdate(cm.namer.ResourceGroupName(), name, req, nil)
	if err != nil {
		return network.VirtualNetwork{}, err
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Virtual network %v created", name))
	return cm.conn.virtualNetworksClient.Get(cm.namer.ResourceGroupName(), name, "")
}

func (cm *clusterManager) getVirtualNetwork() (network.VirtualNetwork, error) {
	return cm.conn.virtualNetworksClient.Get(cm.namer.ResourceGroupName(), cm.namer.VirtualNetworkName(), "")
}

func (cm *clusterManager) createNetworkSecurityGroup() (network.SecurityGroup, error) {
	securityGroupName := cm.namer.NetworkSecurityGroupName()
	securityGroup := network.SecurityGroup{
		Name:     types.StringP(securityGroupName),
		Location: types.StringP(cm.ctx.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(cm.ctx.Name),
		},
	}
	_, err := cm.conn.securityGroupsClient.CreateOrUpdate(cm.namer.ResourceGroupName(), securityGroupName, securityGroup, nil)
	if err != nil {
		return network.SecurityGroup{}, err
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Network security group %v created", securityGroupName))
	return cm.conn.securityGroupsClient.Get(cm.namer.ResourceGroupName(), securityGroupName, "")
}

func (cm *clusterManager) createSubnetID(vn *network.VirtualNetwork, sg *network.SecurityGroup) (network.Subnet, error) {
	name := cm.namer.SubnetName()
	routeTable, err := cm.createRouteTable()
	if err != nil {
		return network.Subnet{}, err
	}
	req := network.Subnet{
		Name: types.StringP(name),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: sg.ID,
			},
			AddressPrefix: types.StringP(cm.ctx.SubnetCidr),
			RouteTable: &network.RouteTable{
				ID: routeTable.ID,
			},
		},
	}

	_, err = cm.conn.subnetsClient.CreateOrUpdate(cm.namer.ResourceGroupName(), *vn.Name, name, req, nil)
	if err != nil {
		return network.Subnet{}, err
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Subnet name %v created", name))
	return cm.conn.subnetsClient.Get(cm.namer.ResourceGroupName(), *vn.Name, name, "")
}

func (cm *clusterManager) getSubnetID(vn *network.VirtualNetwork) (network.Subnet, error) {
	return cm.conn.subnetsClient.Get(cm.namer.ResourceGroupName(), *vn.Name, cm.namer.SubnetName(), "")
}

func (cm *clusterManager) createRouteTable() (network.RouteTable, error) {
	name := cm.namer.RouteTableName()
	req := network.RouteTable{
		Name:     types.StringP(name),
		Location: types.StringP(cm.ctx.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(cm.ctx.Name),
		},
	}
	_, err := cm.conn.routeTablesClient.CreateOrUpdate(cm.namer.ResourceGroupName(), name, req, nil)
	if err != nil {
		return network.RouteTable{}, err
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Route table %v created", name))
	return cm.conn.routeTablesClient.Get(cm.namer.ResourceGroupName(), name, "")
}

func (cm *clusterManager) createNetworkSecurityRule(sg *network.SecurityGroup) error {
	sshRuleName := cm.namer.NetworkSecurityRule("ssh")
	sshRule := network.SecurityRule{
		Name: types.StringP(sshRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access: network.Allow,
			DestinationAddressPrefix: types.StringP("*"),
			DestinationPortRange:     types.StringP("22"),
			Direction:                network.Inbound,
			Priority:                 types.Int32P(100),
			Protocol:                 network.TCP,
			SourceAddressPrefix:      types.StringP("*"),
			SourcePortRange:          types.StringP("*"),
		},
	}
	_, err := cm.conn.securityRulesClient.CreateOrUpdate(cm.namer.ResourceGroupName(), *sg.Name, sshRuleName, sshRule, nil)
	if err != nil {
		return err
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Network security rule %v created", sshRuleName))
	sslRuleName := cm.namer.NetworkSecurityRule("ssl")
	sslRule := network.SecurityRule{
		Name: types.StringP(sshRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access: network.Allow,
			DestinationAddressPrefix: types.StringP("*"),
			DestinationPortRange:     types.StringP("443"),
			Direction:                network.Inbound,
			Priority:                 types.Int32P(110),
			Protocol:                 network.TCP,
			SourceAddressPrefix:      types.StringP("*"),
			SourcePortRange:          types.StringP("*"),
		},
	}
	_, err = cm.conn.securityRulesClient.CreateOrUpdate(cm.namer.ResourceGroupName(), *sg.Name, sslRuleName, sslRule, nil)
	if err != nil {
		return err
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Network security rule %v created", sslRuleName))

	mastersslRuleName := cm.namer.NetworkSecurityRule("masterssl")
	mastersslRule := network.SecurityRule{
		Name: types.StringP(mastersslRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access: network.Allow,
			DestinationAddressPrefix: types.StringP("*"),
			DestinationPortRange:     types.StringP("6443"),
			Direction:                network.Inbound,
			Priority:                 types.Int32P(120),
			Protocol:                 network.TCP,
			SourceAddressPrefix:      types.StringP("*"),
			SourcePortRange:          types.StringP("*"),
		},
	}
	_, err = cm.conn.securityRulesClient.CreateOrUpdate(cm.namer.ResourceGroupName(), *sg.Name, mastersslRuleName, mastersslRule, nil)
	if err != nil {
		return err
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Network security rule %v created", mastersslRuleName))

	return err
}

func (cm *clusterManager) createStorageAccount() (armstorage.Account, error) {
	storageName := cm.ctx.AzureCloudConfig.StorageAccountName
	req := armstorage.AccountCreateParameters{
		Location: types.StringP(cm.ctx.Zone),
		Sku: &armstorage.Sku{
			Name: armstorage.StandardLRS,
		},
		Tags: &map[string]*string{
			"KubernetesCluster": types.StringP(cm.ctx.Name),
		},
	}
	_, err := cm.conn.storageClient.Create(cm.namer.ResourceGroupName(), storageName, req, nil)
	if err != nil {
		return armstorage.Account{}, err
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Storage account %v created", storageName))
	keys, err := cm.conn.storageClient.ListKeys(cm.namer.ResourceGroupName(), storageName)
	if err != nil {
		return armstorage.Account{}, err
	}
	storageClient, err := azstore.NewBasicClient(storageName, *(*(keys.Keys))[0].Value)
	if err != nil {
		return armstorage.Account{}, err
	}

	err = storageClient.GetBlobService().CreateContainer(cm.namer.StorageContainerName(), azstore.ContainerAccessTypeContainer)
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

			nodeScript := im.RenderStartupScript(cm.ctx.NewScriptOptions(), ng.Sku, system.RoleKubernetesPool)
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
			ki.Role = system.RoleKubernetesPool
			cm.ins.Instances = append(cm.ins.Instances, ki)
			// cm.ins.Instances = append(cm.ins.Instances, ki)
		}
*/
