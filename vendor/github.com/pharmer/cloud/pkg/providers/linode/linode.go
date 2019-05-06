package linode

import (
	"context"
	"net/http"

	"github.com/linode/linodego"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
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
