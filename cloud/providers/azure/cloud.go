package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster

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
	cred, err := cloud.Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.Azure{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	namer := namer{cluster: cluster}
	cluster.Spec.Cloud.Azure.CloudConfig = &api.AzureCloudConfig{
		TenantID:           typed.TenantID(),
		SubscriptionID:     typed.SubscriptionID(),
		AadClientID:        typed.ClientID(),
		AadClientSecret:    typed.ClientSecret(),
		ResourceGroup:      namer.ResourceGroupName(),
		Location:           cluster.Spec.Cloud.Zone,
		SubnetName:         namer.SubnetName(),
		SecurityGroupName:  namer.NetworkSecurityGroupName(),
		VnetName:           namer.VirtualNetworkName(),
		RouteTableName:     namer.RouteTableName(),
		StorageAccountName: namer.GenStorageAccountName(),
	}
	cluster.Spec.Cloud.CloudConfigPath = "/etc/kubernetes/azure.json"
	cluster.Spec.Cloud.Azure.StorageAccountName = cluster.Spec.Cloud.Azure.CloudConfig.StorageAccountName

	/*
		if az.Cloud == "" {
			az.Environment = azure.PublicCloud
		} else {
			az.Environment, err = azure.EnvironmentFromName(az.Cloud)
			if err != nil {
				return nil, err
			}
		}
	*/
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
		cluster:                 cluster,
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
