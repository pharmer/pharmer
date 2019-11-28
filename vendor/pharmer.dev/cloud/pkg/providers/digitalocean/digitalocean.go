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
package digitalocean

import (
	"context"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/credential"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

type Client struct {
	Client *godo.Client
	ctx    context.Context
}

func NewClient(opts credential.DigitalOcean) (*Client, error) {
	g := &Client{ctx: context.Background()}
	g.Client = getClient(g.ctx, opts.Token())
	return g, nil
}

func (g *Client) GetName() string {
	return apis.DigitalOcean
}

func (g *Client) GetCredentialFormat() v1.CredentialFormat {
	return credential.DigitalOcean{}.Format()
}

func (g *Client) ListRegions() ([]v1.Region, error) {
	regionList, _, err := g.Client.Regions.List(g.ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	var regions []v1.Region
	for _, region := range regionList {
		r := ParseRegion(&region)
		regions = append(regions, *r)
	}
	return regions, nil
}

//Rgion.Slug is used as zone name
func (g *Client) ListZones() ([]string, error) {
	regionList, _, err := g.Client.Regions.List(g.ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	var zones []string
	for _, region := range regionList {
		zones = append(zones, region.Slug)
	}
	return zones, nil
}

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	sizeList, _, err := g.Client.Sizes.List(g.ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	var instances []v1.MachineType
	for _, s := range sizeList {
		ins, err := ParseMachineType(&s)
		if err != nil {
			return nil, err
		}
		instances = append(instances, *ins)
	}
	return instances, nil
}

func getClient(ctx context.Context, doToken string) *godo.Client {
	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: doToken,
	}))
	return godo.NewClient(oauthClient)
}
