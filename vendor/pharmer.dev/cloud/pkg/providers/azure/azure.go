/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package azure

import (
	"context"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/credential"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/appscode/go/log"
)

type Client struct {
	SubscriptionId string
	GroupsClient   subscriptions.Client
	VmSizesClient  compute.VirtualMachineSizesClient
}

func NewClient(opts credential.Azure) (*Client, error) {
	g := &Client{
		SubscriptionId: opts.SubscriptionID(),
	}
	var err error

	baseURI := azure.PublicCloud.ResourceManagerEndpoint
	config, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, opts.TenantID())
	if err != nil {
		return nil, err
	}

	spt, err := adal.NewServicePrincipalToken(*config, opts.ClientID(), opts.ClientSecret(), baseURI)
	if err != nil {
		return nil, err
	}
	g.GroupsClient = subscriptions.NewClient()
	g.GroupsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	g.VmSizesClient = compute.NewVirtualMachineSizesClient(opts.SubscriptionID())
	g.VmSizesClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Azure
}

func (g *Client) GetCredentialFormat() v1.CredentialFormat {
	return credential.Azure{}.Format()
}

func (g *Client) ListRegions() ([]v1.Region, error) {
	regionList, err := g.GroupsClient.ListLocations(context.Background(), g.SubscriptionId)
	if err != nil {
		return nil, err
	}
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
			log.Infoln(err.Error())
			continue
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
