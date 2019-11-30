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
package scaleway

import (
	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/credential"

	scaleway "github.com/scaleway/scaleway-cli/pkg/api"
)

type Client struct {
	ParClient *scaleway.ScalewayAPI
	AmsClient *scaleway.ScalewayAPI
}

func NewClient(opts credential.Scaleway) (*Client, error) {
	g := &Client{}
	var err error
	g.ParClient, err = scaleway.NewScalewayAPI(opts.Organization(), opts.Token(), "gen-data", "par1")
	if err != nil {
		return nil, err
	}
	g.AmsClient, err = scaleway.NewScalewayAPI(opts.Organization(), opts.Token(), "gen-data", "ams1")
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Scaleway
}

func (g *Client) GetCredentialFormat() v1.CredentialFormat {
	return credential.Scaleway{}.Format()
}

func (g *Client) ListRegions() ([]v1.Region, error) {
	regions := []v1.Region{
		{
			Location: "Paris, France",
			Region:   "par1",
			Zones:    []string{"par1"},
		},
		{
			Location: "Amsterdam, Netherlands",
			Region:   "ams1",
			Zones:    []string{"ams1"},
		},
	}
	return regions, nil
}

func (g *Client) ListZones() ([]string, error) {
	zones := []string{
		"ams1",
		"par1",
	}
	return zones, nil
}

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	instanceList, err := g.ParClient.GetProductsServers()
	if err != nil {
		return nil, err
	}
	var instances []v1.MachineType
	instancePos := map[string]int{}
	for pos, ins := range instanceList.Servers {
		instance, err := ParseInstance(pos, &ins)
		instance.Spec.Zones = []string{"par1"}
		if err != nil {
			return nil, err
		}
		instances = append(instances, *instance)
		instancePos[instance.Spec.SKU] = len(instances) - 1
	}

	instanceList, err = g.AmsClient.GetProductsServers()
	if err != nil {
		return nil, err
	}
	for pos, ins := range instanceList.Servers {
		instance, err := ParseInstance(pos, &ins)
		if err != nil {
			return nil, err
		}
		if index, found := instancePos[instance.Spec.SKU]; found {
			instances[index].Spec.Zones = append(instances[index].Spec.Zones, "ams1")
		} else {
			instance.Spec.Zones = []string{"ams1"}
			instances = append(instances, *instance)
			instancePos[instance.Spec.SKU] = len(instances) - 1
		}
	}

	return instances, nil
}
