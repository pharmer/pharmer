package vultr

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pharmer/cloud/pkg/apis"

	vultr "github.com/JamesClonk/vultr/lib"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	Client *vultr.Client
}

type PlanExtended struct {
	vultr.Plan
	Category   string `json:"plan_type"`
	Deprecated bool   `json:"deprecated"`
}

func NewClient(opts Options) (*Client, error) {
	g := &Client{
		Client: vultr.NewClient(opts.Token, nil),
	}
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Vultr
}

func (g *Client) ListCredentialFormats() []v1.CredentialFormat {
	return []v1.CredentialFormat{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: apis.Vultr,
				Labels: map[string]string{
					"cloud.pharmer.io/provider": apis.Vultr,
				},
				Annotations: map[string]string{
					"cloud.pharmer.io/cluster-credential": "",
					"cloud.pharmer.io/dns-credential":     "",
				},
			},
			Spec: v1.CredentialFormatSpec{
				Provider:      apis.Vultr,
				DisplayFormat: "field",
				Fields: []v1.CredentialField{
					{
						Envconfig: "VULTR_TOKEN",
						Form:      "vultr_token",
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
	regionlist, err := g.Client.GetRegions()
	if err != nil {
		return nil, err
	}
	var regions []v1.Region
	for _, r := range regionlist {
		region := ParseRegion(&r)
		regions = append(regions, *region)
	}
	return regions, nil
}

func (g *Client) ListZones() ([]string, error) {
	regions, err := g.ListRegions()
	if err != nil {
		return nil, err
	}
	var zones []string
	//since we use data.Region.Region as Zone name
	for _, r := range regions {
		zones = append(zones, r.Region)
	}
	return zones, nil
}

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	var instances []v1.MachineType
	planReq, err := g.getPlanRequest()
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(planReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var planList map[string]PlanExtended
	err = json.Unmarshal(body, &planList)
	if err != nil {
		return nil, err
	}
	for _, p := range planList {
		instance, err := ParseInstance(&p)
		if err != nil {
			return nil, err
		}
		instances = append(instances, *instance)
	}
	return instances, nil
}

func (g *Client) getPlanRequest() (*http.Request, error) {
	req, err := http.NewRequest("GET", "https://api.vultr.com/v1/plans/list?type=all", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", g.Client.UserAgent)
	req.Header.Add("Accept", "application/json")
	return req, nil
}
