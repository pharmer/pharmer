package vultr

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	vultr "github.com/JamesClonk/vultr/lib"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
)

type Client struct {
	Data   *VultrData
	Client *vultr.Client
}

type VultrData data.CloudData

type PlanExtended struct {
	vultr.Plan
	Category   string `json:"plan_type"`
	Deprecated bool   `json:"deprecated"`
}

func NewClient(vultrApiToken string) (*Client, error) {
	g := &Client{
		Client: vultr.NewClient(vultrApiToken, nil),
	}
	var err error
	data, err := util.GetDataFormFile("vultr")
	if err != nil {
		return nil, err
	}
	d := VultrData(*data)
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
	regionlist, err := g.Client.GetRegions()
	if err != nil {
		return nil, err
	}
	regions := []data.Region{}
	for _, r := range regionlist {
		region := ParseRegion(&r)
		regions = append(regions, *region)
	}
	return regions, nil
}

func (g *Client) GetZones() ([]string, error) {
	regions, err := g.GetRegions()
	if err != nil {
		return nil, err
	}
	zones := []string{}
	//since we use data.Region.Region as Zone name
	for _, r := range regions {
		zones = append(zones, r.Region)
	}
	return zones, nil
}

func (g *Client) GetInstanceTypes() ([]data.InstanceType, error) {
	instances := []data.InstanceType{}
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
