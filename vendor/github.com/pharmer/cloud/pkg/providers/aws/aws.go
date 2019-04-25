package aws

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pharmer/cloud/pkg/apis"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func NewClient(opts Options) (*Client, error) {
	c := &Client{}
	var err error
	c.session, err = session.NewSession(&aws.Config{
		Region:      &opts.Region,
		Credentials: credentials.NewStaticCredentials(opts.AccessKeyID, opts.SecretAccessKey, ""),
	})
	return c, err
}

func (g *Client) GetName() string {
	return apis.AWS
}

func (g *Client) ListCredentialFormats() []v1.CredentialFormat {
	return []v1.CredentialFormat{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: apis.AWS,
				Labels: map[string]string{
					apis.KeyCloudProvider: apis.AWS,
				},
				Annotations: map[string]string{
					apis.KeyClusterCredential: "",
					apis.KeyDNSCredential:     "",
					apis.KeyStorageCredential: "",
				},
			},
			Spec: v1.CredentialFormatSpec{
				Provider:      apis.AWS,
				DisplayFormat: "field",
				Fields: []v1.CredentialField{
					{
						Envconfig: "AWS_ACCESS_KEY_ID",
						Form:      "aws_access_key_id",
						JSON:      "accessKeyID",
						Label:     "Access Key Id",
						Input:     "text",
					},
					{
						Envconfig: "AWS_SECRET_ACCESS_KEY",
						Form:      "aws_secret_access_key",
						JSON:      "secretAccessKey",
						Label:     "Secret Access Key",
						Input:     "password",
					},
				},
			},
		},
	}
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
