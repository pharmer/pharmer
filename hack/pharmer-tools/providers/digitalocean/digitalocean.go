package digitalocean

import (
	"context"

	"github.com/digitalocean/godo"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
	"golang.org/x/oauth2"
)

type Client struct {
	Data   *DigitalOceanData
	Client *godo.Client
	Ctx    context.Context
}

type DigitalOceanData data.CloudData

func NewClient(doToken string) (*Client, error) {
	g := &Client{
		Ctx:  context.Background(),
		Data: &DigitalOceanData{},
	}
	var err error
	g.Client = getClient(g.Ctx, doToken)

	data, err := util.GetDataFormFile("digitalocean")
	if err != nil {
		return nil, err
	}
	d := DigitalOceanData(*data)
	g.Data = &d
	return g, nil
}

func (g *Client) GetName() string {
	return g.Data.Name
}

func (g *Client) GetEnvs() []string {
	return g.Data.Envs
}

func (g *Client) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *Client) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

func (g *Client) GetRegions() ([]data.Region, error) {
	regionList, _, err := g.Client.Regions.List(g.Ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	regions := []data.Region{}
	for _, region := range regionList {
		r := ParseRegion(&region)
		regions = append(regions, *r)
	}
	return regions, nil
}

//Rgion.Slug is used as zone name
func (g *Client) GetZones() ([]string, error) {
	regionList, _, err := g.Client.Regions.List(g.Ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	zones := []string{}
	for _, region := range regionList {
		zones = append(zones, region.Slug)
	}
	return zones, nil
}

func (g *Client) GetInstanceTypes() ([]data.InstanceType, error) {
	sizeList, _, err := g.Client.Sizes.List(g.Ctx, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	instances := []data.InstanceType{}
	for _, s := range sizeList {
		ins, err := ParseSizes(&s)
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
