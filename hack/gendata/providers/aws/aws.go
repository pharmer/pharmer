package aws

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

type AwsClient struct {
	Data    *AwsData         `json:"data,omitempty"`
	Session *session.Session `json:"session"`
}

type AwsData data.CloudData

type Ec2Instance struct {
	Family        string      `json:"family"`
	Instance_type string      `json:"instance_type"`
	Memory        json.Number `json:"memory"`
	VCPU          json.Number `json:"vCPU"`
	Pricing       interface{} `json:"pricing"`
}

func NewAwsClient(awsRegionName, awsAccessKeyId, awsSecretAccessKey, versions string) (*AwsClient, error) {
	g := &AwsClient{}
	var err error
	g.Session, err = session.NewSession(&aws.Config{
		Region:      &awsRegionName,
		Credentials: credentials.NewStaticCredentials(awsAccessKeyId, awsSecretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}
	data, err := util.GetDataFormFile("aws")
	if err != nil {
		return nil, err
	}
	d := AwsData(*data)
	g.Data = &d
	return g, nil
}

func (g *AwsClient) GetName() string {
	return g.Data.Name
}

func (g *AwsClient) GetEnvs() []string {
	return g.Data.Envs
}

func (g *AwsClient) GetCredentials() []data.CredentialFormat {
	return g.Data.Credentials
}

func (g *AwsClient) GetKubernets() []data.Kubernetes {
	return g.Data.Kubernetes
}

func (g *AwsClient) GetRegions() ([]data.Region, error) {
	//Create new EC2 client
	svc := ec2.New(g.Session)
	regionList, err := svc.DescribeRegions(nil)
	if err != nil {
		return nil, err
	}
	regions := []data.Region{}
	for _, r := range regionList.Regions {
		regions = append(regions, *ParseRegion(r))
	}
	tempSession, err := session.NewSession(&aws.Config{
		Credentials: g.Session.Config.Credentials,
	})
	if err != nil {
		return nil, err
	}
	for pos, region := range regions {
		tempSession.Config.Region = &region.Region
		svc := ec2.New(tempSession)
		zoneList, err := svc.DescribeAvailabilityZones(nil)
		if err != nil {
			return nil, err
		}
		region.Zones = []string{}
		for _, z := range zoneList.AvailabilityZones {
			if *z.RegionName != region.Region {
				return nil, fmt.Errorf("Wrong available zone for %v.", region.Region)
			}
			region.Zones = append(region.Zones, *z.ZoneName)
		}
		regions[pos].Zones = region.Zones
	}
	return regions, nil
}

func (g *AwsClient) GetZones() ([]string, error) {
	visZone := map[string]bool{}
	regionList, err := g.GetRegions()
	if err != nil {
		return nil, err
	}
	zones := []string{}
	for _, r := range regionList {
		for _, z := range r.Zones {
			if _, found := visZone[z]; !found {
				visZone[z] = true
				zones = append(zones, z)
			}
		}
	}
	return zones, nil
}

//https://ec2instances.info/instances.json
//https://github.com/powdahound/ec2instances.info
func (g *AwsClient) GetInstanceTypes() ([]data.InstanceType, error) {

	client := &http.Client{}
	req, err := getInstanceRequest()
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var instanceList []Ec2Instance
	err = json.Unmarshal(body, &instanceList)
	if err != nil {
		return nil, err
	}
	instances := []data.InstanceType{}
	for _, ins := range instanceList {
		i, err := ParseInstance(&ins)
		if err != nil {
			return nil, err
		}
		instances = append(instances, *i)
	}
	return instances, nil
}

func getInstanceRequest() (*http.Request, error) {
	req, err := http.NewRequest("GET", "https://ec2instances.info/instances.json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	return req, nil
}
