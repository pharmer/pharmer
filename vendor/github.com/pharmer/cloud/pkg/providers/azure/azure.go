package azure

import (
	"context"

	"github.com/pharmer/cloud/pkg/apis"

	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	SubscriptionId string
	GroupsClient   subscriptions.Client
	VmSizesClient  compute.VirtualMachineSizesClient
}

func NewClient(opts Options) (*Client, error) {
	g := &Client{
		SubscriptionId: opts.SubscriptionID,
	}
	var err error

	baseURI := azure.PublicCloud.ResourceManagerEndpoint
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, opts.TenantID)
	if err != nil {
		return nil, err
	}

	spt, err := adal.NewServicePrincipalToken(*config, opts.ClientID, opts.ClientSecret, baseURI)
	if err != nil {
		return nil, err
	}
	g.GroupsClient = subscriptions.NewClient()
	g.GroupsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	g.VmSizesClient = compute.NewVirtualMachineSizesClient(opts.SubscriptionID)
	g.VmSizesClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Azure
}

func (g *Client) ListCredentialFormats() []v1.CredentialFormat {
	return []v1.CredentialFormat{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: apis.Azure + "-cred",
				Labels: map[string]string{
					"cloud.pharmer.io/provider": apis.Azure,
				},
				Annotations: map[string]string{
					"cloud.pharmer.io/cluster-credential": "",
					"cloud.pharmer.io/dns-credential":     "",
				},
			},
			Spec: v1.CredentialFormatSpec{
				Provider:      apis.Azure,
				DisplayFormat: "field",
				Fields: []v1.CredentialField{
					{
						Envconfig: "AZURE_TENANT_ID",
						Form:      "azure_tenant_id",
						JSON:      "tenantID",
						Label:     "Tenant Id",
						Input:     "text",
					},
					{
						Envconfig: "AZURE_SUBSCRIPTION_ID",
						Form:      "azure_subscription_id",
						JSON:      "subscriptionID",
						Label:     "Subscription Id",
						Input:     "text",
					},
					{
						Envconfig: "AZURE_CLIENT_ID",
						Form:      "azure_client_id",
						JSON:      "clientID",
						Label:     "Client Id",
						Input:     "text",
					},
					{
						Envconfig: "AZURE_CLIENT_SECRET",
						Form:      "azure_client_secret",
						JSON:      "clientSecret",
						Label:     "Client Secret",
						Input:     "password",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: apis.Azure + "-storage-cred",
				Labels: map[string]string{
					"cloud.pharmer.io/provider": apis.Azure,
				},
				Annotations: map[string]string{
					"cloud.pharmer.io/storage-credential": "",
				},
			},
			Spec: v1.CredentialFormatSpec{
				Provider:      apis.Azure,
				DisplayFormat: "field",
				Fields: []v1.CredentialField{
					{
						Envconfig: "AZURE_STORAGE_ACCOUNT",
						Form:      "azure_storage_account",
						JSON:      "account",
						Label:     "Azure Storage Account",
						Input:     "text",
					},
					{
						Envconfig: "AZURE_STORAGE_KEY",
						Form:      "azure_storage_key",
						JSON:      "key",
						Label:     "Azure Storage Account Key",
						Input:     "password",
					},
				},
			},
		},
	}
}

func (g *Client) ListRegions() ([]v1.Region, error) {
	regionList, err := g.GroupsClient.ListLocations(context.Background(), g.SubscriptionId)
	var regions []v1.Region
	for _, r := range *regionList.Value {
		region := ParseRegion(&r)
		regions = append(regions, *region)
	}
	return regions, err
}

func (g *Client) ListZones() ([]string, error) {
	regions, err := g.ListRegions()
	if err != nil {
		return nil, err
	}
	visZone := map[string]bool{}
	var zones []string
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

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	zones, err := g.ListZones()
	if err != nil {
		return nil, err
	}
	var instances []v1.MachineType
	//to find the position in instances array
	instancePos := map[string]int{}
	for _, zone := range zones {
		instanceList, err := g.VmSizesClient.List(context.Background(), zone)
		if err != nil {
			return nil, err
		}
		for _, ins := range *instanceList.Value {
			instance, err := ParseInstance(&ins)
			if err != nil {
				return nil, err
			}
			pos, found := instancePos[instance.Spec.SKU]
			if found {
				instances[pos].Spec.Zones = append(instances[pos].Spec.Zones, zone)
			} else {
				instancePos[instance.Spec.SKU] = len(instances)
				instance.Spec.Zones = []string{zone}
				instances = append(instances, *instance)
			}
		}
	}
	return instances, nil
}
