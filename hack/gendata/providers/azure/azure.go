package azure

import (
	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/resources/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pharmer/pharmer/data"
)

type AzureClient struct {
	Data           *AzureDefaultData                 `json:"data,omitempty"`
	SubscriptionId string                            `json:"subscription_id"`
	GroupsClient   subscriptions.GroupClient         `json:"groups_client"`
	VmSizesClient  compute.VirtualMachineSizesClient `json:"vm_sizes_client"`
}

type AzureDefaultData struct {
	Name        string                  `json:"name"`
	Envs        []string                `json:"envs,omitempty"`
	Credentials []data.CredentialFormat `json:"credentials"`
	Kubernetes  []data.Kubernetes       `json:"kubernetes"`
}

func NewAzureClient(tenantId, subscriptionId, clientId, clientSecret, versions string) (*AzureClient, error) {
	g := &AzureClient{
		SubscriptionId: subscriptionId,
	}
	var err error
	g.Data, err = GetDefault(versions)
	if err != nil {
		return nil, err
	}

	baseURI := azure.PublicCloud.ResourceManagerEndpoint
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantId)
	if err != nil {
		return nil, err
	}

	spt, err := adal.NewServicePrincipalToken(*config, clientId, clientSecret, baseURI)
	if err != nil {
		return nil, err
	}
	g.GroupsClient = subscriptions.NewGroupClient()
	g.GroupsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	g.VmSizesClient = compute.NewVirtualMachineSizesClient(subscriptionId)
	g.VmSizesClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	return g, nil
}

func (g *AzureClient) GetName() string {
	return g.Data.Name
}

func (g *AzureClient) GetEnvs() []string {
	return g.Data.Envs
}

func (g *AzureClient) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *AzureClient) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

func (g *AzureClient) GetRegions() ([]data.Region, error) {
	regionList, err := g.GroupsClient.ListLocations(g.SubscriptionId)
	regions := []data.Region{}
	for _, r := range *regionList.Value {
		region := ParseRegion(&r)
		regions = append(regions, *region)
	}
	return regions, err
}

func (g *AzureClient) GetZones() ([]string, error) {
	regions, err := g.GetRegions()
	if err != nil {
		return nil, err
	}
	visZone := map[string]bool{}
	zones := []string{}
	for _, r := range regions {
		for _, z := range r.Zones {
			if _, found := visZone[z]; !found {
				zones = append(zones, z)
				visZone[z] = true
			}
		}
	}
	return zones, nil
}

func (g *AzureClient) GetInstanceTypes() ([]data.InstanceType, error) {
	zones, err := g.GetZones()
	if err != nil {
		return nil, err
	}
	instances := []data.InstanceType{}
	//to find the positon in instances array
	instancePos := map[string]int{}
	for _, zone := range zones {
		instanceList, err := g.VmSizesClient.List(zone)
		if err != nil {
			return nil, err
		}
		for _, ins := range *instanceList.Value {
			instance, err := ParseInstance(&ins)
			if err != nil {
				return nil, err
			}
			pos, found := instancePos[instance.SKU]
			if found {
				instances[pos].Zones = append(instances[pos].Zones, zone)
			} else {
				instancePos[instance.SKU] = len(instances)
				instance.Zones = []string{zone}
				instances = append(instances, *instance)
			}
		}
	}
	return instances, nil
}
