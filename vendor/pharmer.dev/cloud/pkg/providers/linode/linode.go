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
package linode

import (
	"context"
	"net/http"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/credential"

	"github.com/linode/linodego"
	"golang.org/x/oauth2"
)

type Client struct {
	Client *linodego.Client
}

func NewClient(opts credential.Linode) (*Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: opts.APIToken()})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	client := linodego.NewClient(oauth2Client)
	g := &Client{
		Client: &client,
	}
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Linode
}

func (g *Client) GetCredentialFormat() v1.CredentialFormat {
	return credential.Linode{}.Format()
}

//DataCenter as region
func (g *Client) ListRegions() ([]v1.Region, error) {
	regionList, err := g.Client.ListRegions(context.Background(), &linodego.ListOptions{})
	if err != nil {
		return nil, err
	}
	var regions []v1.Region
	for _, r := range regionList {
		region := ParseRegion(&r)
		regions = append(regions, *region)
	}
	return regions, nil
}

//data.Region.Region as Zone
func (g *Client) ListZones() ([]string, error) {
	regionList, err := g.ListRegions()
	if err != nil {
		return nil, err
	}
	var zones []string
	for _, r := range regionList {
		zones = append(zones, r.Region)
	}
	return zones, nil
}

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	instanceList, err := g.Client.ListTypes(context.Background(), &linodego.ListOptions{})
	if err != nil {
		return nil, err
	}
	var instances []v1.MachineType
	for _, ins := range instanceList {
		instance, err := ParseInstance(&ins)
		if err != nil {
			return nil, err
		}
		instances = append(instances, *instance)
	}
	return instances, nil
}
