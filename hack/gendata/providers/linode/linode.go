package linode

import (
	"github.com/pharmer/pharmer/data"
	"github.com/taoh/linodego"
)

type LinodeClient struct {
	Data   *LinodeDefaultData `json:"data,omitempty"`
	Client *linodego.Client   `json:"client,omitempty"`
}

type LinodeDefaultData struct {
	Name        string                  `json:"name"`
	Envs        []string                `json:"envs,omitempty"`
	Credentials []data.CredentialFormat `json:"credentials"`
	Kubernetes  []data.Kubernetes       `json:"kubernetes"`
}

func NewLinodeClient(linodeApiKey, versions string) (*LinodeClient, error) {
	g := &LinodeClient{
		Client: linodego.NewClient(linodeApiKey, nil),
	}
	var err error
	g.Data, err = GetDefault(versions)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *LinodeClient) GetName() string {
	return g.Data.Name
}

func (g *LinodeClient) GetEnvs() []string {
	return g.Data.Envs
}

func (g *LinodeClient) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *LinodeClient) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

//DataCenter as region
func (g *LinodeClient) GetRegions() ([]data.Region, error) {
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
func (g *LinodeClient) GetZones() ([]string, error) {
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

func (g *LinodeClient) GetInstanceTypes() ([]data.InstanceType, error) {
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
