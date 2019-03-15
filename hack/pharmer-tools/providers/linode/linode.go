package linode

import (
	"context"
	"net/http"

	"github.com/linode/linodego"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
	"golang.org/x/oauth2"
)

type Client struct {
	Data   *LinodeData
	Client *linodego.Client
}

type LinodeData data.CloudData

func NewClient(linodeApiToken string) (*Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: linodeApiToken})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	client := linodego.NewClient(oauth2Client)
	g := &Client{
		Client: &client,
	}
	var err error
	data, err := util.GetDataFormFile("linode")
	if err != nil {
		return nil, err
	}
	d := LinodeData(*data)
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

//DataCenter as region
func (g *Client) GetRegions() ([]data.Region, error) {
	regionList, err := g.Client.ListRegions(context.Background(), &linodego.ListOptions{})
	if err != nil {
		return nil, err
	}
	regions := []data.Region{}
	for _, r := range regionList {
		region := ParseRegion(&r)
		regions = append(regions, *region)
	}
	return regions, nil
}

//data.Region.Region as Zone
func (g *Client) GetZones() ([]string, error) {
	regionList, err := g.GetRegions()
	if err != nil {
		return nil, err
	}
	zones := []string{}
	for _, r := range regionList {
		zones = append(zones, r.Region)
	}
	return zones, nil
}

func (g *Client) GetInstanceTypes() ([]data.InstanceType, error) {
	instanceList, err := g.Client.ListTypes(context.Background(), &linodego.ListOptions{})
	if err != nil {
		return nil, err
	}
	instances := []data.InstanceType{}
	for _, ins := range instanceList {
		instance, err := ParseInstance(&ins)
		if err != nil {
			return nil, err
		}
		instances = append(instances, *instance)
	}
	return instances, nil
}
