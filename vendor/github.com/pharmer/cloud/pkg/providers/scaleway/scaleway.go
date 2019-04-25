package scaleway

import (
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	scaleway "github.com/scaleway/scaleway-cli/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	ParClient *scaleway.ScalewayAPI
	AmsClient *scaleway.ScalewayAPI
}

func NewClient(opts Options) (*Client, error) {
	g := &Client{}
	var err error
	g.ParClient, err = scaleway.NewScalewayAPI(opts.Organization, opts.Token, "gen-data", "par1")
	if err != nil {
		return nil, err
	}
	g.AmsClient, err = scaleway.NewScalewayAPI(opts.Organization, opts.Token, "gen-data", "ams1")
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Scaleway
}

func (g *Client) ListCredentialFormats() []v1.CredentialFormat {
	return []v1.CredentialFormat{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: apis.Scaleway,
				Labels: map[string]string{
					"cloud.pharmer.io/provider": apis.Scaleway,
				},
				Annotations: map[string]string{
					"cloud.pharmer.io/cluster-credential": "",
				},
			},
			Spec: v1.CredentialFormatSpec{
				Provider:      apis.Scaleway,
				DisplayFormat: "field",
				Fields: []v1.CredentialField{
					{
						Envconfig: "SCALEWAY_ORGANIZATION",
						Form:      "scaleway_organization",
						JSON:      "organization",
						Label:     "Organization",
						Input:     "text",
					},
					{
						Envconfig: "SCALEWAY_TOKEN",
						Form:      "scaleway_token",
						JSON:      "token",
						Label:     "Token",
						Input:     "password",
					},
				},
			},
		},
	}
}

func (g *Client) ListRegions() ([]v1.Region, error) {
	regions := []v1.Region{
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

func (g *Client) ListZones() ([]string, error) {
	zones := []string{
		"ams1",
		"par1",
	}
	return zones, nil
}

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	instanceList, err := g.ParClient.GetProductsServers()
	if err != nil {
		return nil, err
	}
	var instances []v1.MachineType
	instancePos := map[string]int{}
	for pos, ins := range instanceList.Servers {
		instance, err := ParseInstance(pos, &ins)
		instance.Spec.Zones = []string{"par1"}
		if err != nil {
			return nil, err
		}
		instances = append(instances, *instance)
		instancePos[instance.Spec.SKU] = len(instances) - 1
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
		if index, found := instancePos[instance.Spec.SKU]; found {
			instances[index].Spec.Zones = append(instances[index].Spec.Zones, "ams1")
		} else {
			instance.Spec.Zones = []string{"ams1"}
			instances = append(instances, *instance)
			instancePos[instance.Spec.SKU] = len(instances) - 1
		}
	}

	return instances, nil
}
