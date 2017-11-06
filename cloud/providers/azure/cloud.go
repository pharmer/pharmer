package azure

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	armstorage "github.com/Azure/azure-sdk-for-go/arm/storage"
	azstore "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
)

const (
	machineIDTemplate = "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s"
	CloudProviderName = "azure"
)

var providerIDRE = regexp.MustCompile(`^` + CloudProviderName + `://(?:.*)/Microsoft.Compute/virtualMachines/(.+)$`)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	namer   namer

	availabilitySetsClient  compute.AvailabilitySetsClient
	vmClient                compute.VirtualMachinesClient
	vmExtensionsClient      compute.VirtualMachineExtensionsClient
	groupsClient            resources.GroupsClient
	publicIPAddressesClient network.PublicIPAddressesClient
	virtualNetworksClient   network.VirtualNetworksClient
	securityGroupsClient    network.SecurityGroupsClient
	securityRulesClient     network.SecurityRulesClient
	subnetsClient           network.SubnetsClient
	routeTablesClient       network.RouteTablesClient
	interfacesClient        network.InterfacesClient
	storageClient           storage.AccountsClient
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}
	cluster.Spec.Cloud.CCMCredentialName = cred.Name

	baseURI := azure.PublicCloud.ResourceManagerEndpoint

	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, typed.TenantID())
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	spt, err := adal.NewServicePrincipalToken(*config, typed.ClientID(), typed.ClientSecret(), baseURI)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	client := autorest.NewClientWithUserAgent(fmt.Sprintf("Azure-SDK-for-Go/%s", compute.Version()))
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	availabilitySetsClient := compute.AvailabilitySetsClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	vmClient := compute.VirtualMachinesClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	vmExtensionsClient := compute.VirtualMachineExtensionsClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}

	groupsClient := resources.GroupsClient{
		ManagementClient: resources.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	virtualNetworksClient := network.VirtualNetworksClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	publicIPAddressesClient := network.PublicIPAddressesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	securityGroupsClient := network.SecurityGroupsClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	securityRulesClient := network.SecurityRulesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	subnetsClient := network.SubnetsClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	routeTablesClient := network.RouteTablesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}
	interfacesClient := network.InterfacesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}

	storageClient := storage.AccountsClient{
		ManagementClient: storage.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: typed.SubscriptionID(),
		},
	}

	return &cloudConnector{
		cluster: cluster,
		ctx:     ctx,
		availabilitySetsClient:  availabilitySetsClient,
		vmClient:                vmClient,
		vmExtensionsClient:      vmExtensionsClient,
		groupsClient:            groupsClient,
		publicIPAddressesClient: publicIPAddressesClient,
		virtualNetworksClient:   virtualNetworksClient,
		securityGroupsClient:    securityGroupsClient,
		securityRulesClient:     securityRulesClient,
		subnetsClient:           subnetsClient,
		routeTablesClient:       routeTablesClient,
		interfacesClient:        interfacesClient,
		storageClient:           storageClient,
	}, nil
}

func (conn *cloudConnector) detectUbuntuImage() error {
	conn.cluster.Spec.Cloud.OS = "UbuntuServer"
	conn.cluster.Spec.Cloud.InstanceImageProject = "Canonical"
	conn.cluster.Spec.Cloud.InstanceImage = "16.04-LTS"
	conn.cluster.Spec.Cloud.Azure.InstanceImageVersion = "latest"
	return nil
}

func (conn *cloudConnector) getResourceGroup() (bool, error) {
	_, err := conn.groupsClient.Get(conn.namer.ResourceGroupName())
	return err == nil, err
}

func (conn *cloudConnector) ensureResourceGroup() (resources.Group, error) {
	req := resources.Group{
		Name:     StringP(conn.namer.ResourceGroupName()),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	return conn.groupsClient.CreateOrUpdate(conn.namer.ResourceGroupName(), req)
}

func (conn *cloudConnector) getAvailablitySet() (compute.AvailabilitySet, error) {
	return conn.availabilitySetsClient.Get(conn.namer.ResourceGroupName(), conn.namer.AvailablitySetName())
}

func (conn *cloudConnector) ensureAvailablitySet() (compute.AvailabilitySet, error) {
	name := conn.namer.AvailablitySetName()
	req := compute.AvailabilitySet{
		Name:     StringP(name),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	return conn.availabilitySetsClient.CreateOrUpdate(conn.namer.ResourceGroupName(), name, req)
}

func (conn *cloudConnector) getVirtualNetwork() (network.VirtualNetwork, error) {
	return conn.virtualNetworksClient.Get(conn.namer.ResourceGroupName(), conn.namer.VirtualNetworkName(), "")
}

func (conn *cloudConnector) ensureVirtualNetwork() (network.VirtualNetwork, error) {
	name := conn.namer.VirtualNetworkName()
	req := network.VirtualNetwork{
		Name:     StringP(name),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{conn.cluster.Spec.Networking.NonMasqueradeCIDR},
			},
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}

	_, errchan := conn.virtualNetworksClient.CreateOrUpdate(conn.namer.ResourceGroupName(), name, req, nil)
	err := <-errchan
	if err != nil {
		return network.VirtualNetwork{}, err
	}
	Logger(conn.ctx).Infof("Virtual network %v created", name)
	return conn.virtualNetworksClient.Get(conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) getNetworkSecurityGroup() (network.SecurityGroup, error) {
	return conn.securityGroupsClient.Get(conn.namer.ResourceGroupName(), conn.namer.NetworkSecurityGroupName(), "")
}

func (conn *cloudConnector) createNetworkSecurityGroup() (network.SecurityGroup, error) {
	securityGroupName := conn.namer.NetworkSecurityGroupName()
	securityGroup := network.SecurityGroup{
		Name:     StringP(securityGroupName),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	_, errchan := conn.securityGroupsClient.CreateOrUpdate(conn.namer.ResourceGroupName(), securityGroupName, securityGroup, nil)
	err := <-errchan
	if err != nil {
		return network.SecurityGroup{}, err
	}
	Logger(conn.ctx).Infof("Network security group %v created", securityGroupName)
	return conn.securityGroupsClient.Get(conn.namer.ResourceGroupName(), securityGroupName, "")
}

func (conn *cloudConnector) getSubnetID(vn *network.VirtualNetwork) (network.Subnet, error) {
	n := &network.VirtualNetwork{}
	if vn.Name == n.Name {
		return network.Subnet{}, errors.New("Virtualnetwork not found").Err()
	}
	return conn.subnetsClient.Get(conn.namer.ResourceGroupName(), *vn.Name, conn.namer.SubnetName(), "")
}

func (conn *cloudConnector) createSubnetID(vn *network.VirtualNetwork, sg *network.SecurityGroup) (network.Subnet, error) {
	name := conn.namer.SubnetName()
	var routeTable network.RouteTable
	var err error
	if routeTable, err = conn.getRouteTable(); err != nil {
		if routeTable, err = conn.createRouteTable(); err != nil {
			return network.Subnet{}, err
		}
	}
	req := network.Subnet{
		Name: StringP(name),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: sg.ID,
			},
			AddressPrefix: StringP(conn.cluster.Spec.Cloud.Azure.SubnetCIDR),
			RouteTable: &network.RouteTable{
				ID: routeTable.ID,
			},
		},
	}

	_, errchan := conn.subnetsClient.CreateOrUpdate(conn.namer.ResourceGroupName(), *vn.Name, name, req, nil)
	err = <-errchan
	if err != nil {
		return network.Subnet{}, err
	}
	Logger(conn.ctx).Infof("Subnet name %v created", name)
	return conn.subnetsClient.Get(conn.namer.ResourceGroupName(), *vn.Name, name, "")
}

func (conn *cloudConnector) getRouteTable() (network.RouteTable, error) {
	return conn.routeTablesClient.Get(conn.namer.ResourceGroupName(), conn.namer.RouteTableName(), "")
}

func (conn *cloudConnector) createRouteTable() (network.RouteTable, error) {
	name := conn.namer.RouteTableName()
	req := network.RouteTable{
		Name:     StringP(name),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	_, errchan := conn.routeTablesClient.CreateOrUpdate(conn.namer.ResourceGroupName(), name, req, nil)
	err := <-errchan
	if err != nil {
		return network.RouteTable{}, err
	}
	Logger(conn.ctx).Infof("Route table %v created", name)
	return conn.routeTablesClient.Get(conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) getNetworkSecurityRule() (bool, error) {
	_, err := conn.securityRulesClient.Get(conn.namer.ResourceGroupName(), conn.namer.NetworkSecurityGroupName(), conn.namer.NetworkSecurityRule("ssh"))
	if err != nil {
		return false, err
	}
	_, err = conn.securityRulesClient.Get(conn.namer.ResourceGroupName(), conn.namer.NetworkSecurityGroupName(), conn.namer.NetworkSecurityRule("ssl"))
	if err != nil {
		return false, err
	}
	_, err = conn.securityRulesClient.Get(conn.namer.ResourceGroupName(), conn.namer.NetworkSecurityGroupName(), conn.namer.NetworkSecurityRule("masterssl"))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) createNetworkSecurityRule(sg *network.SecurityGroup) error {
	sshRuleName := conn.namer.NetworkSecurityRule("ssh")
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
	_, errchan := conn.securityRulesClient.CreateOrUpdate(conn.namer.ResourceGroupName(), *sg.Name, sshRuleName, sshRule, nil)
	err := <-errchan
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Network security rule %v created", sshRuleName)
	sslRuleName := conn.namer.NetworkSecurityRule("ssl")
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
	_, errchan = conn.securityRulesClient.CreateOrUpdate(conn.namer.ResourceGroupName(), *sg.Name, sslRuleName, sslRule, nil)
	err = <-errchan
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Network security rule %v created", sslRuleName)

	mastersslRuleName := conn.namer.NetworkSecurityRule("masterssl")
	mastersslRule := network.SecurityRule{
		Name: StringP(mastersslRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access: network.SecurityRuleAccessAllow,
			DestinationAddressPrefix: StringP("*"),
			DestinationPortRange:     StringP(fmt.Sprintf("%d", conn.cluster.Spec.API.BindPort)),
			Direction:                network.SecurityRuleDirectionInbound,
			Priority:                 Int32P(120),
			Protocol:                 network.SecurityRuleProtocolTCP,
			SourceAddressPrefix:      StringP("*"),
			SourcePortRange:          StringP("*"),
		},
	}
	_, errchan = conn.securityRulesClient.CreateOrUpdate(conn.namer.ResourceGroupName(), *sg.Name, mastersslRuleName, mastersslRule, nil)
	err = <-errchan
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Network security rule %v created", mastersslRuleName)

	return err
}

func (conn *cloudConnector) getStorageAccount() (armstorage.Account, error) {
	storageName := conn.cluster.Spec.Cloud.Azure.StorageAccountName
	return conn.storageClient.GetProperties(conn.namer.ResourceGroupName(), storageName)
}

func (conn *cloudConnector) createStorageAccount() (armstorage.Account, error) {
	storageName := conn.cluster.Spec.Cloud.Azure.StorageAccountName
	req := armstorage.AccountCreateParameters{
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		Sku: &armstorage.Sku{
			Name: armstorage.StandardLRS,
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	_, errchan := conn.storageClient.Create(conn.namer.ResourceGroupName(), storageName, req, nil)
	err := <-errchan
	if err != nil {
		return armstorage.Account{}, err
	}
	Logger(conn.ctx).Infof("Storage account %v created", storageName)
	keys, err := conn.storageClient.ListKeys(conn.namer.ResourceGroupName(), storageName)
	if err != nil {
		return armstorage.Account{}, err
	}
	storageClient, err := azstore.NewBasicClient(storageName, *(*(keys.Keys))[0].Value)
	if err != nil {
		return armstorage.Account{}, err
	}

	bs := storageClient.GetBlobService()
	_, err = bs.GetContainerReference(conn.namer.StorageContainerName()).CreateIfNotExists(&azstore.CreateContainerOptions{Access: azstore.ContainerAccessTypeContainer})
	if err != nil {
		return armstorage.Account{}, err
	}
	return conn.storageClient.GetProperties(conn.namer.ResourceGroupName(), storageName)
}

func (conn *cloudConnector) createPublicIP(name string, alloc network.IPAllocationMethod) (network.PublicIPAddress, error) {
	req := network.PublicIPAddress{
		Name:     StringP(name),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: alloc,
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}

	_, errchan := conn.publicIPAddressesClient.CreateOrUpdate(conn.namer.ResourceGroupName(), name, req, nil)
	err := <-errchan
	if err != nil {
		return network.PublicIPAddress{}, err
	}
	Logger(conn.ctx).Infof("Public ip addres %v created", name)
	return conn.publicIPAddressesClient.Get(conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) getPublicIP(name string) (network.PublicIPAddress, error) {
	return conn.publicIPAddressesClient.Get(conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) getNetworkInterface(name string) (network.Interface, error) {
	return conn.interfacesClient.Get(conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) GetNodeGroup(instanceGroup string) (bool, map[string]*api.NodeInfo, error) {
	var flag bool = false
	existingNGs := make(map[string]*api.NodeInfo)
	vm, err := conn.vmClient.List(conn.namer.ResourceGroupName())
	if err != nil {
		return false, existingNGs, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	for _, i := range *vm.Value {
		name := *i.Name
		if strings.HasPrefix(name, instanceGroup) {
			flag = true
			nic, _ := conn.getNetworkInterface(conn.namer.NetworkInterfaceName(name))
			pip, _ := conn.getPublicIP(conn.namer.PublicIPName(name))
			instance, err := conn.newKubeInstance(i, nic, pip)
			if err != nil {
				return flag, existingNGs, errors.FromErr(err).WithContext(conn.ctx).Err()
			}
			existingNGs[*i.Name] = instance
		}

	}
	return flag, existingNGs, nil
	//Logger(conn.ctx).Infof("Found virtual machine %v", vm)
}

func (conn *cloudConnector) createNetworkInterface(name string, sg network.SecurityGroup, subnet network.Subnet, alloc network.IPAllocationMethod, internalIP string, pip network.PublicIPAddress) (network.Interface, error) {
	req := network.Interface{
		Name:     StringP(name),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: StringP("ipconfig"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &network.Subnet{
							ID: subnet.ID,
						},
						PrivateIPAllocationMethod: alloc,
						PublicIPAddress: &network.PublicIPAddress{
							ID: pip.ID,
						},
					},
				},
			},
			EnableIPForwarding: TrueP(),
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: sg.ID,
			},
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	if alloc == network.Static {
		if internalIP == "" {
			return network.Interface{}, errors.New("No private IP provided for Static allocation.").WithContext(conn.ctx).Err()
		}
		(*req.IPConfigurations)[0].PrivateIPAddress = StringP(internalIP)
	}
	_, errchan := conn.interfacesClient.CreateOrUpdate(conn.namer.ResourceGroupName(), name, req, nil)
	err := <-errchan
	if err != nil {
		return network.Interface{}, err
	}
	Logger(conn.ctx).Infof("Network interface %v created", name)
	return conn.interfacesClient.Get(conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) getVirtualMachine(name string) (compute.VirtualMachine, error) {
	return conn.vmClient.Get(conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) createVirtualMachine(nic network.Interface, as compute.AvailabilitySet, sa armstorage.Account, vmName, data, vmSize string) (compute.VirtualMachine, error) {
	req := compute.VirtualMachine{
		Name:     StringP(vmName),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			AvailabilitySet: &compute.SubResource{
				ID: as.ID,
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: nic.ID,
					},
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  StringP(vmName),
				AdminPassword: StringP(conn.cluster.Spec.Cloud.Azure.RootPassword),
				AdminUsername: StringP(conn.namer.AdminUsername()),
				CustomData:    StringP(base64.StdEncoding.EncodeToString([]byte(data))),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: BoolP(!_env.FromHost().DebugEnabled()),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								KeyData: StringP(string(SSHKey(conn.ctx).PublicKey)),
								Path:    StringP(fmt.Sprintf("/home/%v/.ssh/authorized_keys", conn.namer.AdminUsername())),
							},
						},
					},
				},
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					Publisher: StringP(conn.cluster.Spec.Cloud.InstanceImageProject),
					Offer:     StringP(conn.cluster.Spec.Cloud.OS),
					Sku:       StringP(conn.cluster.Spec.Cloud.InstanceImage),
					Version:   StringP(conn.cluster.Spec.Cloud.Azure.InstanceImageVersion),
				},
				OsDisk: &compute.OSDisk{
					Caching:      compute.ReadWrite,
					CreateOption: compute.FromImage,
					Name:         StringP(conn.namer.BootDiskName(vmName)),
					Vhd: &compute.VirtualHardDisk{
						URI: StringP(conn.namer.BootDiskURI(sa, vmName)),
					},
				},
			},
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(vmSize),
			},
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}

	_, errchan := conn.vmClient.CreateOrUpdate(conn.namer.ResourceGroupName(), vmName, req, nil)
	err := <-errchan
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	Logger(conn.ctx).Infof("Virtual machine with disk %v password %v created", conn.namer.BootDiskURI(sa, vmName), conn.cluster.Spec.Cloud.Azure.RootPassword)
	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-linux-extensions-customscript?toc=%2fazure%2fvirtual-machines%2flinux%2ftoc.json
	// https://github.com/Azure/custom-script-extension-linux
	// old: https://github.com/Azure/azure-linux-extensions/tree/master/CustomScript
	// https://docs.microsoft.com/en-us/azure/virtual-machines/virtual-machines-windows-classic-inject-custom-data
	Logger(conn.ctx).Infof("Running startup script in virtual machine %v", vmName)
	extName := vmName + "-script"
	extReq := compute.VirtualMachineExtension{
		Name:     StringP(extName),
		Type:     StringP("Microsoft.Compute/virtualMachines/extensions"),
		Location: StringP(conn.cluster.Spec.Cloud.Zone),
		VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
			Publisher:               StringP("Microsoft.Azure.Extensions"),
			Type:                    StringP("CustomScript"),
			TypeHandlerVersion:      StringP("2.0"),
			AutoUpgradeMinorVersion: TrueP(),
			Settings: &map[string]interface{}{
				"commandToExecute": "cat /var/lib/waagent/CustomData | base64 --decode | /bin/bash",
			},
			// ProvisioningState
		},
		Tags: &map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	_, errchan = conn.vmExtensionsClient.CreateOrUpdate(conn.namer.ResourceGroupName(), vmName, extName, extReq, nil)
	err = <-errchan
	if err != nil {
		return compute.VirtualMachine{}, err
	}

	//Logger(conn.ctx).Infof("Restarting virtual machine %v", vmName)
	//_, err = conn.vmClient.Restart(conn.namer.ResourceGroupName(), vmName, nil)
	//if err != nil {
	//	return compute.VirtualMachine{}, err
	//}

	vm, err := conn.vmClient.Get(conn.namer.ResourceGroupName(), vmName, compute.InstanceView)
	Logger(conn.ctx).Infof("Found virtual machine %v", vm)
	return vm, err
}

func (conn *cloudConnector) DeleteVirtualMachine(vmName string) error {
	_, errchan := conn.vmClient.Delete(conn.namer.ResourceGroupName(), vmName, nil)
	err := <-errchan
	if err != nil {
		return err
	}
	storageName := conn.cluster.Spec.Cloud.Azure.StorageAccountName
	keys, err := conn.storageClient.ListKeys(conn.namer.ResourceGroupName(), storageName)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Virtual machine %v deleted", vmName)
	storageClient, err := azstore.NewBasicClient(storageName, *(*(keys.Keys))[0].Value)
	if err != nil {
		return err
	}
	bs := storageClient.GetBlobService()
	_, err = bs.GetContainerReference(storageName).GetBlobReference(conn.namer.BlobName(vmName)).DeleteIfExists(nil)
	return err
}

func (conn *cloudConnector) newKubeInstance(vm compute.VirtualMachine, nic network.Interface, pip network.PublicIPAddress) (*api.NodeInfo, error) {
	// TODO: Load once
	cred, err := Store(conn.ctx).Credentials().Get(conn.cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", conn.cluster.Spec.CredentialName, err)
	}

	i := api.NodeInfo{
		Name:       strings.ToLower(*vm.Name),
		ExternalID: fmt.Sprintf(machineIDTemplate, typed.SubscriptionID(), conn.namer.ResourceGroupName(), *vm.Name),
		PrivateIP:  *(*nic.IPConfigurations)[0].PrivateIPAddress,
	}
	if pip.IPAddress != nil {
		i.PublicIP = *pip.IPAddress
	}
	return &i, nil
}

func (conn *cloudConnector) StartNode(nodeName, token string, as compute.AvailabilitySet, sg network.SecurityGroup, sn network.Subnet, ng *api.NodeGroup) (*api.NodeInfo, error) {
	ki := &api.NodeInfo{}

	nodePIP, err := conn.createPublicIP(conn.namer.PublicIPName(nodeName), network.Dynamic)
	if err != nil {
		return ki, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	nodeNIC, err := conn.createNetworkInterface(conn.namer.NetworkInterfaceName(nodeName), sg, sn, network.Dynamic, "", nodePIP)
	if err != nil {
		return ki, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	sa, err := conn.getStorageAccount()
	if err != nil {
		return ki, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	script, err := conn.renderStartupScript(ng, token)
	if err != nil {
		return ki, err
	}

	nodeVM, err := conn.createVirtualMachine(nodeNIC, as, sa, nodeName, script, ng.Spec.Template.Spec.SKU)
	if err != nil {
		return ki, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	nodePIP, err = conn.getPublicIP(conn.namer.PublicIPName(nodeName))
	if err != nil {
		return ki, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	ki, err = conn.newKubeInstance(nodeVM, nodeNIC, nodePIP)
	if err != nil {
		return &api.NodeInfo{}, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	return ki, nil
}

func (conn *cloudConnector) CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	as, err := conn.getAvailablitySet()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	vn, err := conn.getVirtualNetwork()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	sn, err := conn.getSubnetID(&vn)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}

	sg, err := conn.getNetworkSecurityGroup()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	return conn.StartNode(name, token, as, sg, sn, ng)
}

func (conn *cloudConnector) DeleteInstanceByProviderID(providerID string) error {
	node, err := splitProviderID(providerID)
	if err != nil {
		return err
	}
	err = conn.DeleteVirtualMachine(node)
	if err != nil {
		return err
	}
	err = conn.deleteNodeNetworkInterface(conn.namer.NetworkInterfaceName(node))
	if err != nil {
		return err
	}
	err = conn.deletePublicIp(conn.namer.PublicIPName(node))
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) deleteResourceGroup() error {
	_, errchan := conn.groupsClient.Delete(conn.namer.ResourceGroupName(), make(chan struct{}))
	Logger(conn.ctx).Infof("Resource group %v deleted", conn.namer.ResourceGroupName())
	return <-errchan
}

func (conn *cloudConnector) deleteNodeNetworkInterface(interfaceName string) error {
	_, errchan := conn.interfacesClient.Delete(conn.cluster.Name, interfaceName, make(chan struct{}))
	Logger(conn.ctx).Infof("Node network interface %v deleted", interfaceName)
	return <-errchan
}

func (conn *cloudConnector) deletePublicIp(ipName string) error {
	_, errchan := conn.publicIPAddressesClient.Delete(conn.cluster.Name, ipName, nil)
	Logger(conn.ctx).Infof("Public ip %v deleted", ipName)
	return <-errchan
}
func (conn *cloudConnector) deleteVirtualMachine(machineName string) error {
	_, errchan := conn.vmClient.Delete(conn.cluster.Name, machineName, make(chan struct{}))
	Logger(conn.ctx).Infof("Virtual machine %v deleted", machineName)
	return <-errchan
}

// splitProviderID converts a providerID to a NodeName.
// ref: https://github.com/kubernetes/kubernetes/blob/0f5f82fa44148c36955f704dd4dfc119bad4b03c/pkg/cloudprovider/providers/azure/azure_util.go#L306
func splitProviderID(providerID string) (string, error) {
	matches := providerIDRE.FindStringSubmatch(providerID)
	if len(matches) != 2 {
		return "", errors.New("error splitting providerID")
	}
	return matches[1], nil
}
