/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package aws

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/credential"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
)

type Client struct {
	session *session.Session
}

type Ec2Instance struct {
	Family       string      `json:"family"`
	InstanceType string      `json:"instance_type"`
	Memory       json.Number `json:"memory"`
	VCPU         json.Number `json:"vCPU"`
	Storage      *Ec2Storage `json:"storage"`
}

type Ec2Storage struct {
	Devices                    int  `json:"devices"`
	IncludesSwapPartition      bool `json:"includes_swap_partition"`
	NvmeSsd                    bool `json:"nvme_ssd"`
	Size                       int  `json:"size"`
	Ssd                        bool `json:"ssd"`
	StorageNeedsInitialization bool `json:"storage_needs_initialization"`
	TrimSupport                bool `json:"trim_support"`
}

func NewClient(opts credential.AWS) (*Client, error) {
	c := &Client{}
	var err error
	c.session, err = session.NewSession(&aws.Config{
		Region:      aws.String(opts.Region()),
		Credentials: credentials.NewStaticCredentials(opts.AccessKeyID(), opts.SecretAccessKey(), ""),
	})
	return c, err
}

func (g *Client) GetName() string {
	return apis.AWS
}

func (g *Client) GetCredentialFormat() v1.CredentialFormat {
	return credential.AWS{}.Format()
}

func (g *Client) ListRegions() ([]v1.Region, error) {
	svc := ec2.New(g.session)
	regionList, err := svc.DescribeRegions(nil)
	if err != nil {
		return nil, err
	}
	var regions []v1.Region
	for _, r := range regionList.Regions {
		regions = append(regions, *ParseRegion(r))
	}
	tempSession, err := session.NewSession(&aws.Config{
		Credentials: g.session.Config.Credentials,
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
				return nil, errors.Errorf("Wrong available zone for %v.", region.Region)
			}
			region.Zones = append(region.Zones, *z.ZoneName)
		}
		regions[pos].Zones = region.Zones
	}
	return regions, nil
}

func (g *Client) ListZones() ([]string, error) {
	visZone := map[string]bool{}
	regionList, err := g.ListRegions()
	if err != nil {
		return nil, err
	}
	var zones []string
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
func (g *Client) ListMachineTypes() ([]v1.MachineType, error) {

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
	var instances []v1.MachineType
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
