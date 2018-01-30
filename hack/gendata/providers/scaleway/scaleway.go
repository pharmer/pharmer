package scaleway

import (
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/util"
	scaleway "github.com/scaleway/scaleway-cli/pkg/api"
)

type ScalewayClient struct {
	Data      *ScalewayData         `json:"data,omitempty"`
	ParClient *scaleway.ScalewayAPI `json:"client,omitempty"`
	AmsClient *scaleway.ScalewayAPI `json:"client,omitempty"`
}

type ScalewayData data.CloudData

func NewScalewayClient(scalewayToken, organization, versions string) (*ScalewayClient, error) {
	g := &ScalewayClient{}
	var err error
	g.ParClient, err = scaleway.NewScalewayAPI(organization, scalewayToken, "gen-data", "par1")
	g.AmsClient, err = scaleway.NewScalewayAPI(organization, scalewayToken, "gen-data", "ams1")
	if err != nil {
		return nil, err
	}
	data, err := util.GetDataFormFile("scaleway")
	if err != nil {
		return nil, err
	}
	d := ScalewayData(*data)
	g.Data = &d
	return g, nil
}

func (g *ScalewayClient) GetName() string {
	return g.Data.Name
}

func (g *ScalewayClient) GetEnvs() []string {
	return g.Data.Envs
}

func (g *ScalewayClient) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *ScalewayClient) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

func (g *ScalewayClient) GetRegions() ([]data.Region, error) {
	regions := []data.Region{
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

func (g *ScalewayClient) GetZones() ([]string, error) {
	zones := []string{
		"ams1",
		"par1",
	}
	return zones, nil
}

func (g *ScalewayClient) GetInstanceTypes() ([]data.InstanceType, error) {
	instanceList, err := g.ParClient.GetProductsServers()
	if err != nil {
		return nil, err
	}
	instances := []data.InstanceType{}
	instancePos := map[string]int{}
	for pos, ins := range instanceList.Servers {
		instance, err := ParseInstance(pos, &ins)
		instance.Zones = []string{"par1"}
		if err != nil {
			return nil, err
		}
		instances = append(instances, *instance)
		instancePos[instance.SKU] = len(instances) - 1
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
		if index, found := instancePos[instance.SKU]; found {
			instances[index].Zones = append(instances[index].Zones, "ams1")
		} else {
			instance.Zones = []string{"ams1"}
			instances = append(instances, *instance)
			instancePos[instance.SKU] = len(instances) - 1
		}
	}

	return instances, nil
}
