package linode

import (
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
	"github.com/taoh/linodego"
)

type Client struct {
	Data   *LinodeData
	Client *linodego.Client
}

type LinodeData data.CloudData

func NewClient(linodeApiToken string) (*Client, error) {
	g := &Client{
		Client: linodego.NewClient(linodeApiToken, nil),
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
	regionList, err := g.Client.Avail.DataCenters()
	if err != nil {
		return nil, err
	}
	regions := []data.Region{}
	for _, r := range regionList.DataCenters {
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
	instanceList, err := g.Client.Avail.LinodePlans()
	if err != nil {
		return nil, err
	}
	instances := []data.InstanceType{}
	for _, ins := range instanceList.LinodePlans {
		instance, err := ParseInstance(&ins)
		if err != nil {
			return nil, err
		}
		instances = append(instances, *instance)
	}
	return instances, nil
}
