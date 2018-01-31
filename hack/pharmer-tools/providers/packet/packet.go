package packet

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/packethost/packngo"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/pharmer-tools/util"
)

type PacketClient struct {
	Data   *PacketData     `json:"data,omitempty"`
	Client *packngo.Client `json:"client,omitempty"`
	//required because current packngo.Plan does not contain Zones
	PlanRequest *http.Request `json:"plan_request,omitempty"`
}

type PacketData data.CloudData

type PlanExtended struct {
	packngo.Plan
	Available_in []struct {
		Href string `json:"href"`
	} `json:"available_in"`
}

type PlanExtendedList struct {
	Plans []PlanExtended `json:"plans"`
}

func NewPacketClient(packetApiKey string) (*PacketClient, error) {
	g := &PacketClient{
		Data: &PacketData{},
	}
	var err error
	g.Client = getClient(packetApiKey)

	data, err := util.GetDataFormFile("packet")
	if err != nil {
		return nil, err
	}
	d := PacketData(*data)
	g.Data = &d

	g.PlanRequest, err = getPlanRequest(packetApiKey)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *PacketClient) GetName() string {
	return g.Data.Name
}

func (g *PacketClient) GetEnvs() []string {
	return g.Data.Envs
}

func (g *PacketClient) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *PacketClient) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

func (g *PacketClient) GetRegions() ([]data.Region, error) {
	facilityList, _, err := g.Client.Facilities.List()
	if err != nil {
		return nil, err
	}
	regions := []data.Region{}
	for _, facility := range facilityList {
		region := ParseFacility(&facility)
		regions = append(regions, *region)
	}
	return regions, nil
}

//Facility.Code as zone
func (g *PacketClient) GetZones() ([]string, error) {
	zones := []string{}
	facilityList, _, err := g.Client.Facilities.List()
	if err != nil {
		return nil, err
	}
	for _, facility := range facilityList {
		zones = append(zones, facility.Code)
	}
	return zones, nil
}

func (g *PacketClient) GetInstanceTypes() ([]data.InstanceType, error) {
	//facilityCode maps facility.ID to facility.Code
	facilityCode := map[string]string{}
	facilityList, _, err := g.Client.Facilities.List()
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
	planList := PlanExtendedList{}
	err = json.Unmarshal(body, &planList)
	if err != nil {
		return nil, err
	}
	instances := []data.InstanceType{}
	for _, plan := range planList.Plans {
		if plan.Line == "baremetal" {
			ins, err := ParsePlan(&plan)
			if err != nil {
				return nil, err
			}
			//add zones
			zones := []string{}
			for _, f := range plan.Available_in {
				code, found := facilityCode[GetFacilityIdFromHerf(f.Href)]
				if found {
					zones = append(zones, code)
					//return nil, fmt.Errorf("%v doesn't exit.",f.Href)
				}
			}
			ins.Zones = zones
			instances = append(instances, *ins)
		}
	}
	return instances, nil
}

func getClient(packetToken string) *packngo.Client {
	return packngo.NewClient("", packetToken, nil)
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
