package azure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/resources/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

//https://docs.microsoft.com/en-us/rest/api/compute/virtualmachines/virtualmachines-list-sizes-region
//https://docs.microsoft.com/en-us/rest/api/compute/virtualmachines/virtualmachines-list-sizes-for-resizing

type credInfo struct {
	ClientId       string `json:"clientID"`
	ClientSecret   string `json:"clientSecret"`
	SubscriptionId string `json:"subscriptionID"`
	TenantId       string `json:"tenantID"`
}

func TestRegion(t *testing.T) {
	cred, err := getCredential()
	if err != nil {
		t.Error(err)
	}
	baseURI := azure.PublicCloud.ResourceManagerEndpoint
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, cred.TenantId)
	if err != nil {
		t.Error(err)
	}

	spt, err := adal.NewServicePrincipalToken(*config, cred.ClientId, cred.ClientSecret, baseURI)
	if err != nil {
		t.Error(err)
	}
	//client := autorest.NewClientWithUserAgent(fmt.Sprintf("Azure-SDK-for-Go/%s", compute.Version()))
	//client.Authorizer = autorest.NewBearerAuthorizer(spt)
	groupsClient := subscriptions.NewGroupClient()
	groupsClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	g := Client{
		GroupsClient:   groupsClient,
		SubscriptionId: cred.SubscriptionId,
	}
	r, err := g.GetRegions()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(r)
}

func TestInstances(t *testing.T) {
	cred, err := getCredential()
	if err != nil {
		t.Error(err)
	}
	baseURI := azure.PublicCloud.ResourceManagerEndpoint
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, cred.TenantId)
	if err != nil {
		t.Error(err)
	}

	spt, err := adal.NewServicePrincipalToken(*config, cred.ClientId, cred.ClientSecret, baseURI)
	if err != nil {
		t.Error(err)
	}
	vmSzClient := compute.NewVirtualMachineSizesClient(cred.SubscriptionId)
	vmSzClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	groupsClient := subscriptions.NewGroupClient()
	groupsClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	g := Client{
		VmSizesClient:  vmSzClient,
		GroupsClient:   groupsClient,
		SubscriptionId: cred.SubscriptionId,
	}
	r, err := g.GetInstanceTypes()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(r)
}

func getCredential() (*credInfo, error) {
	var cred credInfo
	bytes, err := ioutil.ReadFile("/home/ac/Downloads/cred/azure.json")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &cred)
	if err != nil {
		return nil, err
	}
	return &cred, nil
}
