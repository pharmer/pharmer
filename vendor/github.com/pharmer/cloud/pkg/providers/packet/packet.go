package packet

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/packethost/packngo"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
)

type Client struct {
	Client *packngo.Client
	//required because current packngo.Plan does not contain Zones
	PlanRequest *http.Request
}

type PlanList struct {
	Plans []packngo.Plan `json:"plans"`
}

func NewClient(opts credential.Packet) (*Client, error) {
	g := &Client{}
	var err error
	g.Client = getClient(opts.APIKey())

	g.PlanRequest, err = getPlanRequest(opts.APIKey())
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *Client) GetName() string {
	return apis.Packet
}

func (g *Client) GetCredentialFormat() v1.CredentialFormat {
	return credential.Packet{}.Format()
}

func (g *Client) ListRegions() ([]v1.Region, error) {
	facilityList, _, err := g.Client.Facilities.List(&packngo.ListOptions{})
	if err != nil {
		return nil, err
	}
	var regions []v1.Region
	for _, facility := range facilityList {
		region := ParseFacility(&facility)
		regions = append(regions, *region)
	}
	return regions, nil
}

//Facility.Code as zone
func (g *Client) ListZones() ([]string, error) {
	var zones []string
	facilityList, _, err := g.Client.Facilities.List(&packngo.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, facility := range facilityList {
		zones = append(zones, facility.Code)
	}
	return zones, nil
}

func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {
	//facilityCode maps facility.ID to facility.Code
	facilityCode := map[string]string{}
	facilityList, _, err := g.Client.Facilities.List(&packngo.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, facility := range facilityList {
		facilityCode[facility.ID] = facility.Code
		//fmt.Println(facility.ID,facility.Code)
	}

	client := &http.Client{}
	resp, err := client.Do(g.PlanRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	planList := PlanList{}
	err = json.Unmarshal(body, &planList)
	if err != nil {
		return nil, err
	}
	var instances []v1.MachineType
	for _, plan := range planList.Plans {
		if plan.Line == "baremetal" {
			ins, err := ParsePlan(&plan)
			if err != nil {
				return nil, err
			}
			//add zones
			var zones []string
			for _, f := range plan.AvailableIn {
				code, found := facilityCode[GetFacilityIdFromHerf(f.URL)]
				if found {
					zones = append(zones, code)
					//return nil, errors.Errorf("%v doesn't exit.",f.Href)
				}
			}
			ins.Spec.Zones = zones
			instances = append(instances, *ins)
		}
	}
	return instances, nil
}

func getClient(packetToken string) *packngo.Client {
	return packngo.NewClientWithAuth("", packetToken, nil)
}

func getPlanRequest(packetToken string) (*http.Request, error) {
	req, err := http.NewRequest("GET", "https://api.packet.net/plans", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Auth-Token", packetToken)
	req.Header.Add("Accept", "application/json")
	return req, nil
}
