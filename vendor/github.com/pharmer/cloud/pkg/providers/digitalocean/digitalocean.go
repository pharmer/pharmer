package digitalocean

import (
	"context"

	"github.com/pharmer/cloud/pkg/apis"

	"github.com/digitalocean/godo"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	Client *godo.Client
	ctx    context.Context
}

func NewClient(opts Options) (*Client, error) {
	g := &Client{ctx: context.Background()}
	g.Client = getClient(g.ctx, opts.Token)
	return g, nil
}

func (g *Client) GetName() string {
	return apis.DigitalOcean
}

func (g *Client) ListCredentialFormats() []v1.CredentialFormat {
	return []v1.CredentialFormat{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: apis.DigitalOcean,
				Labels: map[string]string{
					apis.KeyCloudProvider: apis.DigitalOcean,
				},
				Annotations: map[string]string{
					apis.KeyClusterCredential: "",
					apis.KeyDNSCredential:     "",
				},
			},
			Spec: v1.CredentialFormatSpec{
				Provider:      apis.DigitalOcean,
				DisplayFormat: "field",
				Fields: []v1.CredentialField{
					{

						Envconfig: "DIGITALOCEAN_TOKEN",
						Form:      "digitalocean_token",
						JSON:      "token",
						Label:     "Personal Access Token",
						Input:     "password",
					},
				},
			},
		},
	}
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
