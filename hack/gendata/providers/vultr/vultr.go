package vultr

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	vultr "github.com/JamesClonk/vultr/lib"
	"github.com/pharmer/pharmer/data"
)

type VultrClient struct {
	Data   *VultrDefaultData `json:"data,omitempty"`
	Client *vultr.Client     `json:"client,omitempty"`
}

type VultrDefaultData struct {
	Name        string                  `json:"name"`
	Envs        []string                `json:"envs,omitempty"`
	Credentials []data.CredentialFormat `json:"credentials"`
	Kubernetes  []data.Kubernetes       `json:"kubernetes"`
}

type PlanExtended struct {
	vultr.Plan
	Catagory   string `json:"plan_type"`
	Deprecated bool   `json:"deprecated"`
}

func NewVultrClient(vultrApiKey, versions string) (*VultrClient, error) {
	g := &VultrClient{
		Client: vultr.NewClient(vultrApiKey, nil),
	}
	var err error
	g.Data, err = GetDefault(versions)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *VultrClient) GetName() string {
	return g.Data.Name
}

func (g *VultrClient) GetEnvs() []string {
	return g.Data.Envs
}

func (g *VultrClient) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *VultrClient) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

func (g *VultrClient) GetRegions() ([]data.Region, error) {
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

func (g *VultrClient) GetZones() ([]string, error) {
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

func (g *VultrClient) GetInstanceTypes() ([]data.InstanceType, error) {
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

func (g *VultrClient) getPlanRequest() (*http.Request, error) {
	req, err := http.NewRequest("GET", "https://api.vultr.com/v1/plans/list?type=all", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", g.Client.UserAgent)
	req.Header.Add("Accept", "application/json")
	return req, nil
}
