package vultr

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	vultr "github.com/JamesClonk/vultr/lib"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
)

type Client struct {
	Client *vultr.Client
}

type PlanExtended struct {
	vultr.Plan
	Category   string `json:"plan_type"`
	Deprecated bool   `json:"deprecated"`
}

func NewClient(opts credential.Vultr) (*Client, error) {
	g := &Client{
		Client: vultr.NewClient(opts.Token(), nil),
	}
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Vultr
}

func (g *Client) GetCredentialFormat() v1.CredentialFormat {
	return credential.Vultr{}.Format()
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
