package digitalocean

import (
	"context"

	"github.com/digitalocean/godo"
	"github.com/pharmer/pharmer/data"
	"golang.org/x/oauth2"
)

type DigitalOceanClient struct {
	Data   *DigitalOceanDefaultData `json:"data,omitempty"`
	Client *godo.Client             `json:"client,omitempty"`
	Ctx    context.Context          `json:"ctx,omitempty"`
}

type DigitalOceanDefaultData struct {
	Name        string                  `json:"name"`
	Envs        []string                `json:"envs,omitempty"`
	Credentials []data.CredentialFormat `json:"credentials"`
	Kubernetes  []data.Kubernetes       `json:"kubernetes"`
}

func NewDigitalOceanClient(doToken, versions string) (*DigitalOceanClient, error) {
	g := &DigitalOceanClient{
		Ctx:  context.Background(),
		Data: &DigitalOceanDefaultData{},
	}
	var err error
	g.Client = getClient(g.Ctx, doToken)
	g.Data, err = GetDefault(versions)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *DigitalOceanClient) GetName() string {
	return g.Data.Name
}

func (g *DigitalOceanClient) GetEnvs() []string {
	return g.Data.Envs
}

func (g *DigitalOceanClient) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *DigitalOceanClient) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

func (g *DigitalOceanClient) GetRegions() ([]data.Region, error) {
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
func (g *DigitalOceanClient) GetZones() ([]string, error) {
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

func (g *DigitalOceanClient) GetInstanceTypes() ([]data.InstanceType, error) {
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
