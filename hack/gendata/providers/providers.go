package providers

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/appscode/go/log"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/cmds/options"
	"github.com/pharmer/pharmer/hack/gendata/providers/aws"
	"github.com/pharmer/pharmer/hack/gendata/providers/azure"
	"github.com/pharmer/pharmer/hack/gendata/providers/digitalocean"
	"github.com/pharmer/pharmer/hack/gendata/providers/gce"
	"github.com/pharmer/pharmer/hack/gendata/providers/linode"
	"github.com/pharmer/pharmer/hack/gendata/providers/packet"
	"github.com/pharmer/pharmer/hack/gendata/providers/scaleway"
	"github.com/pharmer/pharmer/hack/gendata/providers/vultr"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

const (
	Gce          string = "gce"
	DigitalOcean string = "digitalocean"
	Packet       string = "packet"
	Aws          string = "aws"
	Azure        string = "azure"
	Vultr        string = "vultr"
	Linode       string = "linode"
	Scaleway     string = "scaleway"
)

type CloudInterface interface {
	GetName() string
	GetEnvs() []string
	GetCredentials() []data.CredentialFormat
	GetKubernets() []data.Kubernetes
	GetRegions() ([]data.Region, error)
	GetZones() ([]string, error)
	GetInstanceTypes() ([]data.InstanceType, error)
}

func NewCloudProvider(opts *options.CloudData) (CloudInterface, error) {
	switch opts.Provider {
	case Gce:
		return gce.NewGceClient(opts.GCEProjectID, opts.CredentialFile, opts.KubernetesVersions)
		break
	case DigitalOcean:
		return digitalocean.NewDigitalOceanClient(opts.DoToken, opts.KubernetesVersions)
		break
	case Packet:
		return packet.NewPacketClient(opts.PacketApiKey, opts.KubernetesVersions)
		break
	case Aws:
		return aws.NewAwsClient(opts.AWSRegion, opts.AWSAccessKeyID, opts.AWSSecretAccessKey, opts.KubernetesVersions)
		break
	case Azure:
		return azure.NewAzureClient(opts.AzureTenantId, opts.AzureSubscriptionId, opts.AzureClientId, opts.AzureClientSecret, opts.KubernetesVersions)
		break
	case Vultr:
		return vultr.NewVultrClient(opts.VultrApiToken, opts.KubernetesVersions)
		break
	case Linode:
		return linode.NewLinodeClient(opts.LinodeApiToken, opts.KubernetesVersions)
		break
	case Scaleway:
		return scaleway.NewScalewayClient(opts.ScalewayToken, opts.ScalewayOrganization, opts.KubernetesVersions)
		break
	default:
		return nil, fmt.Errorf("Valid/Supported provider name required")
	}
	return nil, nil
}

//get data from api
func GetCloudData(cloudInterface CloudInterface) (*data.CloudData, error) {
	var err error
	cloudData := data.CloudData{
		Name:        cloudInterface.GetName(),
		Envs:        cloudInterface.GetEnvs(),
		Credentials: cloudInterface.GetCredentials(),
		Kubernetes:  cloudInterface.GetKubernets(),
	}
	cloudData.Regions, err = cloudInterface.GetRegions()
	if err != nil {
		return nil, err
	}
	cloudData.InstanceTypes, err = cloudInterface.GetInstanceTypes()
	if err != nil {
		return nil, err
	}
	return &cloudData, nil
}

//write data in [path to pharmer]/data/files/[provider]/
func WriteCloudData(cloudData *data.CloudData, fileName string) error {
	dataBytes, err := json.MarshalIndent(cloudData, "", "  ")
	dir, err := util.GetWriteDir()
	if err != nil {
		return err
	}
	err = util.WriteFile(filepath.Join(dir, cloudData.Name, fileName), dataBytes)
	return err
}

//region merge rule:
//	if region doesn't exist in old data, but exists in new data, then add it
//	if region exist in both, then
//		if field data exists in both new and old data , then take the new data
//		otherwise, take data from (old or new)whichever contains it
//
// instanceType merge rule: same as region rule
//
//In MergeCloudData, we merge only the region and instanceType data
func MergeCloudData(old, new *data.CloudData) (*data.CloudData, error) {
	//region merge
	regionIndex := map[string]int{} //keep regionName,corresponding region index in old.Regions[] as (key,value) pair
	for index, r := range old.Regions {
		regionIndex[r.Region] = index
	}
	for index, _ := range new.Regions {
		pos, found := regionIndex[new.Regions[index].Region]
		if found {
			//location
			if new.Regions[index].Location == "" && old.Regions[pos].Location != "" {
				new.Regions[index].Location = old.Regions[pos].Location
			}
			//zones
			if len(new.Regions[index].Zones) == 0 && len(old.Regions[pos].Zones) != 0 {
				new.Regions[index].Location = old.Regions[pos].Location
			}
		}
	}

	//instanceType
	instanceIndex := map[string]int{} //keep SKU,corresponding instance index in old.InstanceTypes[] as (key,value) pair
	for index, ins := range old.InstanceTypes {
		instanceIndex[ins.SKU] = index
	}
	for index, _ := range new.InstanceTypes {
		pos, found := instanceIndex[new.InstanceTypes[index].SKU]
		if found {
			//description
			if new.InstanceTypes[index].Description == "" && old.InstanceTypes[pos].Description != "" {
				new.InstanceTypes[index].Description = old.InstanceTypes[pos].Description
			}
			//zones
			if len(new.InstanceTypes[index].Zones) == 0 && len(old.InstanceTypes[pos].Zones) == 0 {
				new.InstanceTypes[index].Zones = old.InstanceTypes[pos].Zones
			}
			//regions
			//if len(new.InstanceTypes[index].Regions)==0 && len(old.InstanceTypes[pos].Regions)!=0 {
			//	new.InstanceTypes[index].Regions = old.InstanceTypes[pos].Regions
			//}
			//Disk
			if new.InstanceTypes[index].Disk == 0 && old.InstanceTypes[pos].Disk != 0 {
				new.InstanceTypes[index].Disk = old.InstanceTypes[pos].Disk
			}
			//RAM
			if new.InstanceTypes[index].RAM == nil && old.InstanceTypes[pos].RAM != nil {
				new.InstanceTypes[index].RAM = old.InstanceTypes[pos].RAM
			}
			//catagory
			if new.InstanceTypes[index].Category == "" && old.InstanceTypes[pos].Category != "" {
				new.InstanceTypes[index].Category = old.InstanceTypes[pos].Category
			}
			//CPU
			if new.InstanceTypes[index].CPU == 0 && old.InstanceTypes[pos].CPU != 0 {
				new.InstanceTypes[index].CPU = old.InstanceTypes[pos].CPU
			}
			//to detect it already added to new
			instanceIndex[new.InstanceTypes[index].SKU] = -1
		}
	}
	for _, index := range instanceIndex {
		if index > -1 {
			//using regions as zones
			if len(old.InstanceTypes[index].Regions) > 0 {
				if len(old.InstanceTypes[index].Zones) == 0 {
					old.InstanceTypes[index].Zones = old.InstanceTypes[index].Regions
				}
				old.InstanceTypes[index].Regions = nil
			}
			new.InstanceTypes = append(new.InstanceTypes, old.InstanceTypes[index])
			new.InstanceTypes[len(new.InstanceTypes)-1].Deprecated = true
		}
	}
	return new, nil
}

//get data from api , merge it with previous data and write the data
//previous data written in cloud_old.json
func MergeAndWriteCloudData(cloudInterface CloudInterface) error {
	log.Infof("Getting cloud data for `%v` provider", cloudInterface.GetName())
	new, err := GetCloudData(cloudInterface)
	if err != nil {
		return err
	}

	old, err := util.GetDataFormFile(cloudInterface.GetName())
	if err != nil {
		return err
	}
	log.Info("Merging cloud data...")
	res, err := MergeCloudData(old, new)
	if err != nil {
		return err
	}

	//err = WriteCloudData(old,"cloud_old.json")
	//if err!=nil {
	//	return err
	//}
	log.Info("Writing cloud data...")
	err = WriteCloudData(res, "cloud.json")
	if err != nil {
		return err
	}
	return nil
}
