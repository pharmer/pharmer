package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/credential"
)

type cloudConnector struct {
	ctx *contexts.ClusterContext

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

func NewConnector(ctx *contexts.ClusterContext) (*cloudConnector, error) {
	subscriptionID, ok := ctx.CloudCredential[credential.AzureSubscriptionID]
	if !ok {
		return nil, errors.New("Missing", credential.AzureSubscriptionID).WithContext(ctx).Err()
	}

	tenantID, ok := ctx.CloudCredential[credential.AzureTenantID]
	if !ok {
		return nil, errors.New("Missing", credential.AzureTenantID).WithContext(ctx).Err()
	}

	clientID, ok := ctx.CloudCredential[credential.AzureClientID]
	if !ok {
		return nil, errors.New("Missing", credential.AzureClientID).WithContext(ctx).Err()
	}

	clientSecret, ok := ctx.CloudCredential[credential.AzureClientSecret]
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

	config, err := azure.PublicCloud.OAuthConfigForTenant(tenantID)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	spt, err := azure.NewServicePrincipalToken(*config, clientID, clientSecret, baseURI)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}

	client := autorest.NewClientWithUserAgent(fmt.Sprintf("Azure-SDK-for-Go/%s", compute.Version()))
	client.Authorizer = spt

	availabilitySetsClient := compute.AvailabilitySetsClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     compute.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	vmClient := compute.VirtualMachinesClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     compute.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	vmExtensionsClient := compute.VirtualMachineExtensionsClient{
		ManagementClient: compute.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     compute.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}

	groupsClient := resources.GroupsClient{
		ManagementClient: resources.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     resources.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	virtualNetworksClient := network.VirtualNetworksClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     network.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	publicIPAddressesClient := network.PublicIPAddressesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     network.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	securityGroupsClient := network.SecurityGroupsClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     network.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	securityRulesClient := network.SecurityRulesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     network.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	subnetsClient := network.SubnetsClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     network.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	routeTablesClient := network.RouteTablesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     network.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}
	interfacesClient := network.InterfacesClient{
		ManagementClient: network.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     network.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}

	storageClient := storage.AccountsClient{
		ManagementClient: storage.ManagementClient{
			Client:         client,
			BaseURI:        baseURI,
			APIVersion:     storage.APIVersion,
			SubscriptionID: subscriptionID,
		},
	}

	return &cloudConnector{
		ctx: ctx,
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
