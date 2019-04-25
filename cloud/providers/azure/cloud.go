package azure

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	. "github.com/appscode/go/types"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	capiAzure "github.com/pharmer/pharmer/apis/v1beta1/azure"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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

	owner string

	availabilitySetsClient  compute.AvailabilitySetsClient
	vmClient                compute.VirtualMachinesClient
	vmExtensionsClient      compute.VirtualMachineExtensionsClient
	groupsClient            resources.GroupsClient
	publicIPAddressesClient network.PublicIPAddressesClient
	virtualNetworksClient   network.VirtualNetworksClient
	securityGroupsClient    network.SecurityGroupsClient
	securityRulesClient     network.SecurityRulesClient
	subnetsClient           network.SubnetsClient
	lbClient                network.LoadBalancersClient
	routeTablesClient       network.RouteTablesClient
	interfacesClient        network.InterfacesClient
	storageClient           storage.AccountsClient
}

func NewConnector(cm *ClusterManager) (*cloudConnector, error) {
	cred, err := Store(cm.ctx).Owner(cm.owner).Credentials().Get(cm.cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cm.cluster.Spec.Config.CredentialName)
	}

	baseURI := azure.PublicCloud.ResourceManagerEndpoint
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, typed.TenantID())
	if err != nil {
		return nil, errors.Wrap(err, ID(cm.ctx))
	}

	spt, err := adal.NewServicePrincipalToken(*config, typed.ClientID(), typed.ClientSecret(), baseURI)
	if err != nil {
		return nil, errors.Wrap(err, ID(cm.ctx))
	}

	client := autorest.NewClientWithUserAgent(fmt.Sprintf("Azure-SDK-for-Go/%s", compute.Version()))
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	availabilitySetsClient := compute.NewAvailabilitySetsClientWithBaseURI(baseURI, typed.SubscriptionID())
	availabilitySetsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	vmClient := compute.NewVirtualMachinesClientWithBaseURI(baseURI, typed.SubscriptionID())
	vmClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	vmExtensionsClient := compute.NewVirtualMachineExtensionsClientWithBaseURI(baseURI, typed.SubscriptionID())
	vmExtensionsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	groupsClient := resources.NewGroupsClientWithBaseURI(baseURI, typed.SubscriptionID())
	groupsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	virtualNetworksClient := network.NewVirtualNetworksClientWithBaseURI(baseURI, typed.SubscriptionID())
	virtualNetworksClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	publicIPAddressesClient := network.NewPublicIPAddressesClientWithBaseURI(baseURI, typed.SubscriptionID())
	publicIPAddressesClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	securityGroupsClient := network.NewSecurityGroupsClientWithBaseURI(baseURI, typed.SubscriptionID())
	securityGroupsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	securityRulesClient := network.NewSecurityRulesClientWithBaseURI(baseURI, typed.SubscriptionID())
	securityRulesClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	subnetsClient := network.NewSubnetsClientWithBaseURI(baseURI, typed.SubscriptionID())
	subnetsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	routeTablesClient := network.NewRouteTablesClientWithBaseURI(baseURI, typed.SubscriptionID())
	routeTablesClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	interfacesClient := network.NewInterfacesClientWithBaseURI(baseURI, typed.SubscriptionID())
	interfacesClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	storageClient := storage.NewAccountsClientWithBaseURI(baseURI, typed.SubscriptionID())
	storageClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	lbClient := network.NewLoadBalancersClientWithBaseURI(baseURI, typed.SubscriptionID())
	lbClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &cloudConnector{
		cluster:                 cm.cluster,
		ctx:                     cm.ctx,
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
		lbClient:                lbClient,

		owner: cm.owner,
		namer: cm.namer,
	}, nil
}

func PrepareCloud(cm *ClusterManager) error {
	var err error

	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
		return err
	}
	if cm.ctx, err = LoadEtcdCertificate(cm.ctx, cm.cluster, cm.owner); err != nil {
		return err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return err
	}
	if cm.ctx, err = LoadSaKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return err
	}

	if cm.conn, err = NewConnector(cm); err != nil {
		return err
	}

	return nil
}

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	if err := CreateNamespace(kc, "azure-provider-system"); err != nil {
		return err
	}

	if err := CreateSecret(kc, "azure-provider-azure-controller-secrets", "azure-provider-system", map[string][]byte{
		"client-id":       []byte(data["clientID"]),
		"client-secret":   []byte(data["clientSecret"]),
		"subscription-id": []byte(data["subscriptionID"]),
		"tenant-id":       []byte(data["tenantID"]),
	}); err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) detectUbuntuImage() error {
	conn.cluster.Spec.Config.Cloud.OS = "UbuntuServer"
	conn.cluster.Spec.Config.Cloud.InstanceImageProject = "Canonical"
	conn.cluster.Spec.Config.Cloud.InstanceImage = "16.04-LTS"
	conn.cluster.Spec.Config.Cloud.Azure.InstanceImageVersion = "latest"
	return nil
}

func (conn *cloudConnector) getResourceGroup() (bool, error) {
	providerConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return false, err
	}
	_, err = conn.groupsClient.Get(context.TODO(), providerConfig.ResourceGroup)
	return err == nil, err
}

func (conn *cloudConnector) ensureResourceGroup() (resources.Group, error) {
	providerConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return resources.Group{}, err
	}
	req := resources.Group{
		Name:     StringP(providerConfig.ResourceGroup),
		Location: StringP(providerConfig.Location),
		Tags: map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}
	return conn.groupsClient.CreateOrUpdate(context.TODO(), conn.namer.ResourceGroupName(), req)
}

func (conn *cloudConnector) getVirtualNetwork() (network.VirtualNetwork, error) {
	return conn.virtualNetworksClient.Get(context.TODO(), conn.namer.ResourceGroupName(), conn.namer.VirtualNetworkName(), "")
}

// ref: https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/b55f066db9cb0f96e81257150a2468942d3996ba/pkg/cloud/azure/services/virtualnetworks/virtualnetworks.go#L51
func (conn *cloudConnector) ensureVirtualNetwork() (network.VirtualNetwork, error) {
	providerConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return network.VirtualNetwork{}, err
	}
	name := conn.namer.GenerateVnetName()
	req := network.VirtualNetwork{
		Name:     StringP(name),
		Location: StringP(providerConfig.Location),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{conn.cluster.Spec.Config.Cloud.Azure.VPCCIDR},
			},
		},
		Tags: map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}

	f, err := conn.virtualNetworksClient.CreateOrUpdate(context.TODO(), providerConfig.ResourceGroup, name, req)
	if err != nil {
		return network.VirtualNetwork{}, err
	}

	if err := f.WaitForCompletionRef(context.Background(), conn.virtualNetworksClient.Client); err != nil {
		return network.VirtualNetwork{}, err
	}

	vn, err := f.Result(conn.virtualNetworksClient)
	if err != nil {
		return network.VirtualNetwork{}, err
	}

	Logger(conn.ctx).Infof("Virtual network %v created", name)
	return vn, nil
}

func (conn *cloudConnector) getNetworkSecurityGroup(name string) (network.SecurityGroup, error) {
	return conn.securityGroupsClient.Get(context.TODO(), conn.namer.ResourceGroupName(), name, "")
}

// ref: https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/b55f066db9cb0f96e81257150a2468942d3996ba/pkg/cloud/azure/services/securitygroups/securitygroups.go#L51
func (conn *cloudConnector) createNetworkSecurityGroup(isControlPlane bool) (network.SecurityGroup, error) {
	providerConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return network.SecurityGroup{}, err
	}

	securityRules := &[]network.SecurityRule{}

	if isControlPlane {
		securityRules = &[]network.SecurityRule{
			{
				Name: to.StringPtr("allow_6443"),
				SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
					Protocol:                 network.SecurityRuleProtocolTCP,
					SourceAddressPrefix:      to.StringPtr("*"),
					SourcePortRange:          to.StringPtr("*"),
					DestinationAddressPrefix: to.StringPtr("*"),
					DestinationPortRange:     to.StringPtr("6443"),
					Access:                   network.SecurityRuleAccessAllow,
					Direction:                network.SecurityRuleDirectionInbound,
					Priority:                 to.Int32Ptr(100),
				},
			},
		}
	}

	var name string
	if isControlPlane {
		name = conn.namer.GenerateControlPlaneSecurityGroupName()
	} else {
		name = conn.namer.GenerateNodeSecurityGroupName()
	}

	f, err := conn.securityGroupsClient.CreateOrUpdate(
		context.Background(),
		conn.namer.ResourceGroupName(),
		name,
		network.SecurityGroup{
			Location: StringP(providerConfig.Location),
			SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
				SecurityRules: securityRules,
			},
			Tags: map[string]*string{
				"KubernetesCluster": StringP(conn.cluster.Name),
			},
		},
	)
	if err != nil {
		return network.SecurityGroup{}, errors.Wrapf(err, "failed to create security group %s in resource group %s", name, conn.namer.ResourceGroupName())
	}

	err = f.WaitForCompletionRef(context.Background(), conn.securityGroupsClient.Client)
	if err != nil {
		return network.SecurityGroup{}, errors.Wrap(err, "cannot create, future response")
	}

	sg, err := f.Result(conn.securityGroupsClient)
	if err != nil {
		return network.SecurityGroup{}, errors.Wrap(err, "result error")
	}
	Logger(conn.ctx).Infof("created security group %s", name)
	return sg, nil
}

func (conn *cloudConnector) getSubnetID(name string) (network.Subnet, error) {
	return conn.subnetsClient.Get(context.TODO(), conn.namer.ResourceGroupName(), conn.namer.GenerateVnetName(), name, "")
}

// ref: https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/c4896544d32792f06f5302c4fd9d2b4fdff358e1/pkg/cloud/azure/services/subnets/subnets.go#L56
func (conn *cloudConnector) createSubnetID(name string, sg *network.SecurityGroup, rt *network.RouteTable) (network.Subnet, error) {
	subnetProperties := network.SubnetPropertiesFormat{
		RouteTable:           rt,
		NetworkSecurityGroup: sg,
	}

	// route table is nil for control plane subnet
	// using this logic for setting up name and cidrs
	if rt == nil {
		subnetProperties.AddressPrefix = &conn.cluster.Spec.Config.Cloud.Azure.ControlPlaneSubnetCIDR
	} else {
		subnetProperties.AddressPrefix = &conn.cluster.Spec.Config.Cloud.Azure.NodeSubnetCIDR
	}

	log.Infof("creating subnet %s in vnet %s", name, conn.namer.GenerateVnetName())
	f, err := conn.subnetsClient.CreateOrUpdate(
		context.Background(),
		conn.namer.ResourceGroupName(),
		conn.namer.GenerateVnetName(),
		name,
		network.Subnet{
			Name:                   to.StringPtr(name),
			SubnetPropertiesFormat: &subnetProperties,
		},
	)
	if err != nil {
		return network.Subnet{}, errors.Wrapf(err, "failed to create subnet %s in resource group %s", name, conn.namer.ResourceGroupName())
	}

	err = f.WaitForCompletionRef(context.Background(), conn.subnetsClient.Client)
	if err != nil {
		return network.Subnet{}, errors.Wrap(err, "cannot create, future response")
	}

	sn, err := f.Result(conn.subnetsClient)
	if err != nil {
		return network.Subnet{}, errors.Wrap(err, "result error")
	}
	log.Infof("successfully created subnet %s in vnet %s", name, conn.namer.GenerateVnetName())
	return sn, nil
}

func (conn *cloudConnector) findLoadBalancer(name string) (network.LoadBalancer, error) {
	return conn.lbClient.Get(context.TODO(), conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) createInternalLoadBalancer(lbName string, subnet *network.Subnet) (network.LoadBalancer, error) {
	clusterConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return network.LoadBalancer{}, err
	}

	probeName := "tcpHTTPSProbe"
	frontEndIPConfigName := "controlplane-internal-lbFrontEnd"
	backEndAddressPoolName := "controlplane-internal-backEndPool"
	idPrefix := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers", conn.cluster.Spec.Config.Cloud.Azure.SubscriptionID, conn.namer.ResourceGroupName())

	future, err := conn.lbClient.CreateOrUpdate(context.Background(),
		clusterConfig.ResourceGroup,
		lbName,
		network.LoadBalancer{
			Sku:      &network.LoadBalancerSku{Name: network.LoadBalancerSkuNameStandard},
			Location: to.StringPtr(clusterConfig.Location),
			LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
					{
						Name: &frontEndIPConfigName,
						FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: network.Static,
							Subnet:                    subnet,
							PrivateIPAddress:          &conn.cluster.Spec.Config.Cloud.Azure.InternalLBIPAddress,
						},
					},
				},
				BackendAddressPools: &[]network.BackendAddressPool{
					{
						Name: &backEndAddressPoolName,
					},
				},
				Probes: &[]network.Probe{
					{
						Name: &probeName,
						ProbePropertiesFormat: &network.ProbePropertiesFormat{
							Protocol:          network.ProbeProtocolTCP,
							Port:              to.Int32Ptr(6443),
							IntervalInSeconds: to.Int32Ptr(15),
							NumberOfProbes:    to.Int32Ptr(4),
						},
					},
				},
				LoadBalancingRules: &[]network.LoadBalancingRule{
					{
						Name: to.StringPtr("LBRuleHTTPS"),
						LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
							Protocol:             network.TransportProtocolTCP,
							FrontendPort:         to.Int32Ptr(6443),
							BackendPort:          to.Int32Ptr(6443),
							IdleTimeoutInMinutes: to.Int32Ptr(4),
							EnableFloatingIP:     to.BoolPtr(false),
							LoadDistribution:     network.LoadDistributionDefault,
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/frontendIPConfigurations/%s", idPrefix, lbName, frontEndIPConfigName)),
							},
							BackendAddressPool: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/backendAddressPools/%s", idPrefix, lbName, backEndAddressPoolName)),
							},
							Probe: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/probes/%s", idPrefix, lbName, probeName)),
							},
						},
					},
				},
			},
		})

	if err != nil {
		return network.LoadBalancer{}, errors.Wrap(err, "cannot create load balancer")
	}

	err = future.WaitForCompletionRef(context.Background(), conn.lbClient.Client)
	if err != nil {
		return network.LoadBalancer{}, errors.Wrap(err, "cannot get internal load balancer create or update future response")
	}

	lb, err := future.Result(conn.lbClient)
	if err != nil {
		return network.LoadBalancer{}, err
	}
	log.Infof("successfully created internal load balancer %s", lbName)
	return lb, nil
}

func (conn *cloudConnector) createPublicLoadBalancer(pip *network.PublicIPAddress) (network.LoadBalancer, error) {
	clusterConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return network.LoadBalancer{}, err
	}

	probeName := "tcpHTTPSProbe"
	frontEndIPConfigName := "controlplane-lbFrontEnd"
	backEndAddressPoolName := "controlplane-backEndPool"
	idPrefix := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers", conn.cluster.Spec.Config.Cloud.Azure.SubscriptionID, clusterConfig.ResourceGroup)
	lbName := conn.namer.GeneratePublicLBName()
	log.Infof("creating public load balancer %s", lbName)

	f, err := conn.lbClient.CreateOrUpdate(
		context.Background(),
		clusterConfig.ResourceGroup,
		lbName,
		network.LoadBalancer{
			Sku:      &network.LoadBalancerSku{Name: network.LoadBalancerSkuNameStandard},
			Location: to.StringPtr(clusterConfig.Location),
			LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
					{
						Name: &frontEndIPConfigName,
						FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: network.Dynamic,
							PublicIPAddress:           pip,
						},
					},
				},
				BackendAddressPools: &[]network.BackendAddressPool{
					{
						Name: &backEndAddressPoolName,
					},
				},
				Probes: &[]network.Probe{
					{
						Name: &probeName,
						ProbePropertiesFormat: &network.ProbePropertiesFormat{
							Protocol:          network.ProbeProtocolTCP,
							Port:              to.Int32Ptr(6443),
							IntervalInSeconds: to.Int32Ptr(15),
							NumberOfProbes:    to.Int32Ptr(4),
						},
					},
				},
				LoadBalancingRules: &[]network.LoadBalancingRule{
					{
						Name: to.StringPtr("LBRuleHTTPS"),
						LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
							Protocol:             network.TransportProtocolTCP,
							FrontendPort:         to.Int32Ptr(6443),
							BackendPort:          to.Int32Ptr(6443),
							IdleTimeoutInMinutes: to.Int32Ptr(4),
							EnableFloatingIP:     to.BoolPtr(false),
							LoadDistribution:     network.LoadDistributionDefault,
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/frontendIPConfigurations/%s", idPrefix, lbName, frontEndIPConfigName)),
							},
							BackendAddressPool: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/backendAddressPools/%s", idPrefix, lbName, backEndAddressPoolName)),
							},
							Probe: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/probes/%s", idPrefix, lbName, probeName)),
							},
						},
					},
				},
				InboundNatRules: &[]network.InboundNatRule{
					{
						Name: to.StringPtr("natRule1"),
						InboundNatRulePropertiesFormat: &network.InboundNatRulePropertiesFormat{
							Protocol:             network.TransportProtocolTCP,
							FrontendPort:         to.Int32Ptr(22),
							BackendPort:          to.Int32Ptr(22),
							EnableFloatingIP:     to.BoolPtr(false),
							IdleTimeoutInMinutes: to.Int32Ptr(4),
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/frontendIPConfigurations/%s", idPrefix, lbName, frontEndIPConfigName)),
							},
						},
					},
					{
						Name: to.StringPtr("natRule2"),
						InboundNatRulePropertiesFormat: &network.InboundNatRulePropertiesFormat{
							Protocol:             network.TransportProtocolTCP,
							FrontendPort:         to.Int32Ptr(2201),
							BackendPort:          to.Int32Ptr(22),
							EnableFloatingIP:     to.BoolPtr(false),
							IdleTimeoutInMinutes: to.Int32Ptr(4),
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/frontendIPConfigurations/%s", idPrefix, lbName, frontEndIPConfigName)),
							},
						},
					},
					{
						Name: to.StringPtr("natRule3"),
						InboundNatRulePropertiesFormat: &network.InboundNatRulePropertiesFormat{
							Protocol:             network.TransportProtocolTCP,
							FrontendPort:         to.Int32Ptr(2202),
							BackendPort:          to.Int32Ptr(22),
							EnableFloatingIP:     to.BoolPtr(false),
							IdleTimeoutInMinutes: to.Int32Ptr(4),
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/frontendIPConfigurations/%s", idPrefix, lbName, frontEndIPConfigName)),
							},
						},
					},
				},
			},
		})

	if err != nil {
		return network.LoadBalancer{}, errors.Wrap(err, "cannot create public load balancer")
	}

	err = f.WaitForCompletionRef(context.Background(), conn.lbClient.Client)
	if err != nil {
		return network.LoadBalancer{}, errors.Wrapf(err, "cannot get public load balancer create or update future response")
	}

	return f.Result(conn.lbClient)
}

func (conn *cloudConnector) getRouteTable() (network.RouteTable, error) {
	return conn.routeTablesClient.Get(context.TODO(), conn.namer.ResourceGroupName(), conn.namer.GenerateNodeRouteTableName(), "")
}

// ref: https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/b55f066db9cb0f96e81257150a2468942d3996ba/pkg/cloud/azure/services/routetables/routetables.go#L51
func (conn *cloudConnector) createRouteTable() (network.RouteTable, error) {
	providerConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return network.RouteTable{}, err
	}

	name := conn.namer.GenerateNodeRouteTableName()

	f, err := conn.routeTablesClient.CreateOrUpdate(
		context.Background(),
		conn.namer.ResourceGroupName(),
		name,
		network.RouteTable{
			Location:                   to.StringPtr(providerConfig.Location),
			RouteTablePropertiesFormat: &network.RouteTablePropertiesFormat{},
			Tags: map[string]*string{
				"KubernetesCluster": StringP(conn.cluster.Name),
			},
		},
	)
	if err != nil {
		return network.RouteTable{}, errors.Wrapf(err, "failed to create route table %s in resource group %s", name, conn.namer.ResourceGroupName())
	}

	err = f.WaitForCompletionRef(context.Background(), conn.routeTablesClient.Client)
	if err != nil {
		return network.RouteTable{}, errors.Wrap(err, "cannot create, future response")
	}

	rt, err := f.Result(conn.routeTablesClient)
	if err != nil {
		return network.RouteTable{}, errors.Wrap(err, "result error")
	}
	Logger(conn.ctx).Infof("successfully created route table %s", name)
	return rt, nil
}

func (conn *cloudConnector) createPublicIP(ipName string) (network.PublicIPAddress, error) {
	log.Infof("Creating public ip %q", ipName)
	clusterConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return network.PublicIPAddress{}, err
	}

	f, err := conn.publicIPAddressesClient.CreateOrUpdate(
		context.Background(),
		clusterConfig.ResourceGroup,
		ipName,
		network.PublicIPAddress{
			Sku:      &network.PublicIPAddressSku{Name: network.PublicIPAddressSkuNameStandard},
			Name:     to.StringPtr(ipName),
			Location: to.StringPtr(clusterConfig.Location),
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
				DNSSettings: &network.PublicIPAddressDNSSettings{
					DomainNameLabel: to.StringPtr(strings.ToLower(ipName)),
					Fqdn:            to.StringPtr(conn.namer.GenerateFQDN(ipName, DefaultAzureDNSZone)),
				},
			},
		},
	)

	if err != nil {
		return network.PublicIPAddress{}, errors.Wrap(err, "cannot create public ip")
	}

	err = f.WaitForCompletionRef(context.Background(), conn.publicIPAddressesClient.Client)
	if err != nil {
		return network.PublicIPAddress{}, errors.Wrap(err, "cannot create, future response")
	}

	publicIP, err := f.Result(conn.publicIPAddressesClient)
	if err != nil {
		return network.PublicIPAddress{}, errors.Wrap(err, "result error")
	}

	log.Infof("Successfully created public ip %q", *publicIP.Name)
	return publicIP, err
}

func (conn *cloudConnector) getPublicIP(name string) (network.PublicIPAddress, error) {
	return conn.publicIPAddressesClient.Get(context.TODO(), conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) getNetworkInterface(name string) (network.Interface, error) {
	return conn.interfacesClient.Get(context.TODO(), conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) createNetworkInterface(name string, subnet *network.Subnet, publicLB, internalLB *network.LoadBalancer) (network.Interface, error) {
	clusterConfig, err := capiAzure.ClusterConfigFromProviderSpec(conn.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return network.Interface{}, err
	}

	nicConfig := &network.InterfaceIPConfigurationPropertiesFormat{}
	nicConfig.Subnet = &network.Subnet{ID: subnet.ID}
	nicConfig.PrivateIPAllocationMethod = network.Dynamic

	backendAddressPools := []network.BackendAddressPool{}

	backendAddressPools = append(backendAddressPools,
		network.BackendAddressPool{
			ID: (*publicLB.BackendAddressPools)[0].ID,
		})

	backendAddressPools = append(backendAddressPools,
		network.BackendAddressPool{
			ID: (*internalLB.BackendAddressPools)[0].ID,
		},
	)

	nicConfig.LoadBalancerBackendAddressPools = &backendAddressPools

	f, err := conn.interfacesClient.CreateOrUpdate(context.Background(),
		clusterConfig.ResourceGroup,
		name,
		network.Interface{
			Location: to.StringPtr(clusterConfig.Location),
			InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
				IPConfigurations: &[]network.InterfaceIPConfiguration{
					{
						Name:                                     to.StringPtr("pipConfig"),
						InterfaceIPConfigurationPropertiesFormat: nicConfig,
					},
				},
			},
		})

	if err != nil {
		return network.Interface{}, errors.Wrapf(err, "failed to create network interface %s in resource group %s", name, clusterConfig.ResourceGroup)
	}

	err = f.WaitForCompletionRef(context.Background(), conn.interfacesClient.Client)
	if err != nil {
		return network.Interface{}, errors.Wrap(err, "cannot create, future response")
	}

	return f.Result(conn.interfacesClient)
}

func (conn *cloudConnector) getVirtualMachine(name string) (compute.VirtualMachine, error) {
	return conn.vmClient.Get(context.TODO(), conn.namer.ResourceGroupName(), name, "")
}

func (conn *cloudConnector) createVirtualMachine(nic network.Interface, vmName, data string, machine *clusterapi.Machine) (compute.VirtualMachine, error) {
	providerConf, err := capiAzure.MachineSpecFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	req := compute.VirtualMachine{
		Name:     StringP(vmName),
		Location: StringP(providerConf.Location),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: nic.ID,
					},
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  StringP(vmName),
				AdminPassword: StringP(conn.cluster.Spec.Config.Cloud.Azure.RootPassword),
				AdminUsername: StringP(conn.namer.AdminUsername()),
				CustomData:    StringP(base64.StdEncoding.EncodeToString([]byte(data))),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: BoolP(!Env(conn.ctx).DebugEnabled()),
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
					Publisher: StringP(providerConf.Image.Publisher),
					Offer:     StringP(providerConf.Image.Offer),
					Sku:       StringP(providerConf.Image.SKU),
					Version:   StringP(providerConf.Image.Version),
				},
				OsDisk: &compute.OSDisk{
					Name:         to.StringPtr(fmt.Sprintf("%s_OSDisk", vmName)),
					OsType:       compute.OperatingSystemTypes(providerConf.OSDisk.OSType),
					CreateOption: compute.DiskCreateOptionTypesFromImage,
					DiskSizeGB:   to.Int32Ptr(providerConf.OSDisk.DiskSizeGB),
					ManagedDisk: &compute.ManagedDiskParameters{
						StorageAccountType: compute.StorageAccountTypes(providerConf.OSDisk.ManagedDisk.StorageAccountType),
					},
				},
			},
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(providerConf.VMSize),
			},
		},
		Tags: map[string]*string{
			"KubernetesCluster": StringP(conn.cluster.Name),
		},
	}

	_, err = conn.vmClient.CreateOrUpdate(context.TODO(), conn.namer.ResourceGroupName(), vmName, req)
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	Logger(conn.ctx).Infof("Running startup script in virtual machine %v", vmName)

	vmextName := "startupScript"

	_, err = conn.vmExtensionsClient.CreateOrUpdate(
		context.Background(),
		conn.namer.ResourceGroupName(),
		vmName,
		vmextName,
		compute.VirtualMachineExtension{
			Name:     StringP(vmextName),
			Location: StringP(providerConf.Location),
			VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
				Type:                    to.StringPtr("CustomScript"),
				TypeHandlerVersion:      to.StringPtr("2.0"),
				AutoUpgradeMinorVersion: to.BoolPtr(true),
				Settings:                map[string]bool{"skipDos2Unix": true},
				Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
				ProtectedSettings:       map[string]string{"script": base64.StdEncoding.EncodeToString([]byte(data))},
			},
			Tags: map[string]*string{
				"KubernetesCluster": StringP(conn.cluster.Name),
			},
		})
	if err != nil {
		return compute.VirtualMachine{}, err
	}

	vm, err := conn.vmClient.Get(context.TODO(), conn.namer.ResourceGroupName(), vmName, compute.InstanceView)
	Logger(conn.ctx).Infof("Found virtual machine %v", vm)
	return vm, err
}

func (conn *cloudConnector) deleteResourceGroup() error {
	_, err := conn.groupsClient.Delete(context.TODO(), conn.namer.ResourceGroupName())
	Logger(conn.ctx).Infof("Resource group %v deleted", conn.namer.ResourceGroupName())
	return err
}
