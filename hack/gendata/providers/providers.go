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
	cloudData = util.SortCloudData(cloudData)
	dataBytes, err := json.MarshalIndent(cloudData, "", "  ")
	dir, err := util.GetWriteDir()
	if err != nil {
		return err
	}
	err = util.WriteFile(filepath.Join(dir, cloudData.Name, fileName), dataBytes)
	return err
}

//region merge rule:
//	if region doesn't exist in old data, but exists in cur data, then add it
//	if region exists in old data, but doesn't exists in cur data, then delete it
//	if region exist in both, then
//		if field data exists in both cur and old data , then take the cur data
//		otherwise, take data from (old or cur)whichever contains it
//
// instanceType merge rule: same as region rule, except
//		if instance exists in old data, but doesn't exists in cur data, then add it , set the deprecated true
//
//In MergeCloudData, we merge only the region and instanceType data
func MergeCloudData(oldData, curData *data.CloudData) (*data.CloudData, error) {
	//region merge
	regionIndex := map[string]int{} //keep regionName,corresponding region index in oldData.Regions[] as (key,value) pair
	for index, r := range oldData.Regions {
		regionIndex[r.Region] = index
	}
	for index, _ := range curData.Regions {
		pos, found := regionIndex[curData.Regions[index].Region]
		if found {
			//location
			if curData.Regions[index].Location == "" && oldData.Regions[pos].Location != "" {
				curData.Regions[index].Location = oldData.Regions[pos].Location
			}
			//zones
			if len(curData.Regions[index].Zones) == 0 && len(oldData.Regions[pos].Zones) != 0 {
				curData.Regions[index].Location = oldData.Regions[pos].Location
			}
		}
	}

	//instanceType
	instanceIndex := map[string]int{} //keep SKU,corresponding instance index in oldData.InstanceTypes[] as (key,value) pair
	for index, ins := range oldData.InstanceTypes {
		instanceIndex[ins.SKU] = index
	}
	for index, _ := range curData.InstanceTypes {
		pos, found := instanceIndex[curData.InstanceTypes[index].SKU]
		if found {
			//description
			if curData.InstanceTypes[index].Description == "" && oldData.InstanceTypes[pos].Description != "" {
				curData.InstanceTypes[index].Description = oldData.InstanceTypes[pos].Description
			}
			//zones
			if len(curData.InstanceTypes[index].Zones) == 0 && len(oldData.InstanceTypes[pos].Zones) == 0 {
				curData.InstanceTypes[index].Zones = oldData.InstanceTypes[pos].Zones
			}
			//regions
			//if len(curData.InstanceTypes[index].Regions)==0 && len(oldData.InstanceTypes[pos].Regions)!=0 {
			//	curData.InstanceTypes[index].Regions = oldData.InstanceTypes[pos].Regions
			//}
			//Disk
			if curData.InstanceTypes[index].Disk == 0 && oldData.InstanceTypes[pos].Disk != 0 {
				curData.InstanceTypes[index].Disk = oldData.InstanceTypes[pos].Disk
			}
			//RAM
			if curData.InstanceTypes[index].RAM == nil && oldData.InstanceTypes[pos].RAM != nil {
				curData.InstanceTypes[index].RAM = oldData.InstanceTypes[pos].RAM
			}
			//catagory
			if curData.InstanceTypes[index].Category == "" && oldData.InstanceTypes[pos].Category != "" {
				curData.InstanceTypes[index].Category = oldData.InstanceTypes[pos].Category
			}
			//CPU
			if curData.InstanceTypes[index].CPU == 0 && oldData.InstanceTypes[pos].CPU != 0 {
				curData.InstanceTypes[index].CPU = oldData.InstanceTypes[pos].CPU
			}
			//to detect it already added to curData
			instanceIndex[curData.InstanceTypes[index].SKU] = -1
		}
	}
	for _, index := range instanceIndex {
		if index > -1 {
			//using regions as zones
			if len(oldData.InstanceTypes[index].Regions) > 0 {
				if len(oldData.InstanceTypes[index].Zones) == 0 {
					oldData.InstanceTypes[index].Zones = oldData.InstanceTypes[index].Regions
				}
				oldData.InstanceTypes[index].Regions = nil
			}
			curData.InstanceTypes = append(curData.InstanceTypes, oldData.InstanceTypes[index])
			curData.InstanceTypes[len(curData.InstanceTypes)-1].Deprecated = true
		}
	}
	return curData, nil
}

//get data from api , merge it with previous data and write the data
//previous data written in cloud_old.json
func MergeAndWriteCloudData(cloudInterface CloudInterface) error {
	log.Infof("Getting cloud data for `%v` provider", cloudInterface.GetName())
	curData, err := GetCloudData(cloudInterface)
	if err != nil {
		return err
	}

	oldData, err := util.GetDataFormFile(cloudInterface.GetName())
	if err != nil {
		return err
	}
	log.Info("Merging cloud data...")
	res, err := MergeCloudData(oldData, curData)
	if err != nil {
		return err
	}

	//err = WriteCloudData(oldData,"cloud_old.json")
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
