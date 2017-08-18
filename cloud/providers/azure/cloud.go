package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/context"
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
	subscriptionID, ok := cluster.Spec.CloudCredential[credential.AzureSubscriptionID]
	if !ok {
		return nil, errors.New("Missing", credential.AzureSubscriptionID).WithContext(ctx).Err()
	}

	tenantID, ok := cluster.Spec.CloudCredential[credential.AzureTenantID]
	if !ok {
		return nil, errors.New("Missing", credential.AzureTenantID).WithContext(ctx).Err()
	}

	clientID, ok := cluster.Spec.CloudCredential[credential.AzureClientID]
	if !ok {
		return nil, errors.New("Missing", credential.AzureClientID).WithContext(ctx).Err()
	}

	clientSecret, ok := cluster.Spec.CloudCredential[credential.AzureClientSecret]
	if !ok {
		return nil, errors.New("Missing", credential.AzureClientSecret).WithContext(ctx).Err()
	}

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

	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	spt, err := adal.NewServicePrincipalToken(*config, clientID, clientSecret, baseURI)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	client := autorest.NewClientWithUserAgent(fmt.Sprintf("Azure-SDK-for-Go/%s", compute.Version()))
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	availabilitySetsClient := compute.AvailabilitySetsClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	vmClient := compute.VirtualMachinesClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	vmExtensionsClient := compute.VirtualMachineExtensionsClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}

	groupsClient := resources.GroupsClient{
		ManagementClient: resources.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	virtualNetworksClient := network.VirtualNetworksClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	publicIPAddressesClient := network.PublicIPAddressesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	securityGroupsClient := network.SecurityGroupsClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	securityRulesClient := network.SecurityRulesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	subnetsClient := network.SubnetsClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	routeTablesClient := network.RouteTablesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}
	interfacesClient := network.InterfacesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
		},
	}

	storageClient := storage.AccountsClient{
		ManagementClient: storage.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			SubscriptionID: subscriptionID,
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
