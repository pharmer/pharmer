package providers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/appscode/go/log"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/cmds/options"
	"github.com/pharmer/cloud/pkg/providers/aws"
	"github.com/pharmer/cloud/pkg/providers/azure"
	"github.com/pharmer/cloud/pkg/providers/digitalocean"
	"github.com/pharmer/cloud/pkg/providers/gce"
	"github.com/pharmer/cloud/pkg/providers/linode"
	"github.com/pharmer/cloud/pkg/providers/packet"
	"github.com/pharmer/cloud/pkg/providers/scaleway"
	"github.com/pharmer/cloud/pkg/providers/vultr"
	"github.com/pharmer/cloud/pkg/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	mu "kmodules.xyz/client-go/meta"
)

var providers = []string{
	apis.GCE,
	apis.DigitalOcean,
	apis.Packet,
	apis.AWS,
	apis.Azure,
	apis.Vultr,
	apis.Linode,
	apis.Scaleway,
}

func List() []string {
	return append([]string(nil), providers...)
}

type Interface interface {
	GetName() string
	ListCredentialFormats() []v1.CredentialFormat
	ListRegions() ([]v1.Region, error)
	ListZones() ([]string, error)
	ListMachineTypes() ([]v1.MachineType, error)
}

func NewCloudProvider(opts Options) (Interface, error) {
	switch opts.Provider {
	case apis.GCE:
		return gce.NewClient(opts.GCE)
	case apis.DigitalOcean:
		return digitalocean.NewClient(opts.Do)
	case apis.Packet:
		return packet.NewClient(opts.Packet)
	case apis.AWS:
		return aws.NewClient(opts.AWS)
	case apis.Azure:
		return azure.NewClient(opts.Azure)
	case apis.Vultr:
		return vultr.NewClient(opts.Vultr)
	case apis.Linode:
		return linode.NewClient(opts.Linode)
	case apis.Scaleway:
		return scaleway.NewClient(opts.Scaleway)
	}
	return nil, errors.Errorf("Unknown cloud provider: %s", opts.Provider)
}

//get data from api
func GetCloudProvider(i Interface) (*v1.CloudProvider, error) {
	var err error
	data := v1.CloudProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.GetName(),
		},
		Spec: v1.CloudProviderSpec{
			CredentialFormats: i.ListCredentialFormats(),
		},
	}
	data.Spec.Regions, err = i.ListRegions()
	if err != nil {
		return nil, err
	}
	data.Spec.MachineTypes, err = i.ListMachineTypes()
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func WriteObject(obj runtime.Object) error {
	kind := mu.GetKind(obj)
	resource := strings.ToLower(kind) + "s"
	name, err := meta.NewAccessor().Name(obj)
	if err != nil {
		return err
	}

	yamlDir := filepath.Join(apis.DataDir, "yaml", "apis", v1.SchemeGroupVersion.Group, v1.SchemeGroupVersion.Version, resource)
	err = os.MkdirAll(yamlDir, 0755)
	if err != nil {
		return err
	}
	jsonDir := filepath.Join(apis.DataDir, "json", "apis", v1.SchemeGroupVersion.Group, v1.SchemeGroupVersion.Version, resource)
	err = os.MkdirAll(jsonDir, 0755)
	if err != nil {
		return err
	}

	yamlBytes, err := mu.MarshalToYAML(obj, v1.SchemeGroupVersion)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(yamlDir, name+".yaml"), yamlBytes, 0755)
	if err != nil {
		return err
	}

	jsonBytes, err := mu.MarshalToPrettyJson(obj, v1.SchemeGroupVersion)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(jsonDir, name+".json"), jsonBytes, 0755)
}

func WriteCloudProvider(data *v1.CloudProvider) error {
	data = util.SortCloudProvider(data)
	err := WriteObject(data)
	if err != nil {
		return err
	}

	for _, mt := range data.Spec.MachineTypes {
		err := WriteObject(&mt)
		if err != nil {
			return err
		}
	}

	for _, mt := range data.Spec.CredentialFormats {
		err := WriteObject(&mt)
		if err != nil {
			return err
		}
	}

	for _, mt := range data.Spec.KubernetesVersions {
		err := WriteObject(&mt)
		if err != nil {
			return err
		}
	}

	return nil
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
//In MergeCloudProvider, we merge only the region and instanceType data
func MergeCloudProvider(oldData, curData *v1.CloudProvider) (*v1.CloudProvider, error) {
	//region merge
	regionIndex := map[string]int{} //keep regionName,corresponding region index in oldData.Regions[] as (key,value) pair
	for index, r := range oldData.Spec.Regions {
		regionIndex[r.Region] = index
	}
	for index := range curData.Spec.Regions {
		pos, found := regionIndex[curData.Spec.Regions[index].Region]
		if found {
			//location
			if curData.Spec.Regions[index].Location == "" && oldData.Spec.Regions[pos].Location != "" {
				curData.Spec.Regions[index].Location = oldData.Spec.Regions[pos].Location
			}
			//zones
			if len(curData.Spec.Regions[index].Zones) == 0 && len(oldData.Spec.Regions[pos].Zones) != 0 {
				curData.Spec.Regions[index].Location = oldData.Spec.Regions[pos].Location
			}
		}
	}

	//instanceType
	instanceIndex := map[string]int{} //keep SKU,corresponding instance index in oldData.MachineTypes[] as (key,value) pair
	for index, ins := range oldData.Spec.MachineTypes {
		instanceIndex[ins.Spec.SKU] = index
	}
	for index := range curData.Spec.MachineTypes {
		pos, found := instanceIndex[curData.Spec.MachineTypes[index].Spec.SKU]
		if found {
			//description
			if curData.Spec.MachineTypes[index].Spec.Description == "" && oldData.Spec.MachineTypes[pos].Spec.Description != "" {
				curData.Spec.MachineTypes[index].Spec.Description = oldData.Spec.MachineTypes[pos].Spec.Description
			}
			//zones
			if len(curData.Spec.MachineTypes[index].Spec.Zones) == 0 && len(oldData.Spec.MachineTypes[pos].Spec.Zones) == 0 {
				curData.Spec.MachineTypes[index].Spec.Zones = oldData.Spec.MachineTypes[pos].Spec.Zones
			}
			//regions
			//if len(curData.Spec.MachineTypes[index].Spec.Regions)==0 && len(oldData.Spec.MachineTypes[pos].Spec.Regions)!=0 {
			//	curData.Spec.MachineTypes[index].Spec.Regions = oldData.Spec.MachineTypes[pos].Spec.Regions
			//}
			//Disk
			if curData.Spec.MachineTypes[index].Spec.Disk == nil && oldData.Spec.MachineTypes[pos].Spec.Disk != nil {
				curData.Spec.MachineTypes[index].Spec.Disk = oldData.Spec.MachineTypes[pos].Spec.Disk
			}
			//RAM
			if curData.Spec.MachineTypes[index].Spec.RAM == nil && oldData.Spec.MachineTypes[pos].Spec.RAM != nil {
				curData.Spec.MachineTypes[index].Spec.RAM = oldData.Spec.MachineTypes[pos].Spec.RAM
			}
			//category
			if curData.Spec.MachineTypes[index].Spec.Category == "" && oldData.Spec.MachineTypes[pos].Spec.Category != "" {
				curData.Spec.MachineTypes[index].Spec.Category = oldData.Spec.MachineTypes[pos].Spec.Category
			}
			//CPU
			if curData.Spec.MachineTypes[index].Spec.CPU == nil && oldData.Spec.MachineTypes[pos].Spec.CPU != nil {
				curData.Spec.MachineTypes[index].Spec.CPU = oldData.Spec.MachineTypes[pos].Spec.CPU
			}
			//to detect it already added to curData
			instanceIndex[curData.Spec.MachineTypes[index].Spec.SKU] = -1
		}
	}
	for _, index := range instanceIndex {
		if index > -1 {
			//using regions as zones
			if len(oldData.Spec.MachineTypes[index].Spec.Regions) > 0 {
				if len(oldData.Spec.MachineTypes[index].Spec.Zones) == 0 {
					oldData.Spec.MachineTypes[index].Spec.Zones = oldData.Spec.MachineTypes[index].Spec.Regions
				}
				oldData.Spec.MachineTypes[index].Spec.Regions = nil
			}
			curData.Spec.MachineTypes = append(curData.Spec.MachineTypes, oldData.Spec.MachineTypes[index])
			curData.Spec.MachineTypes[len(curData.Spec.MachineTypes)-1].Spec.Deprecated = true
		}
	}
	return curData, nil
}

//get data from api , merge it with previous data and write the data
//previous data written in cloud_old.json
func MergeAndWriteCloudProvider(i Interface) error {
	log.Infof("Getting cloud data for `%v` provider", i.GetName())
	curData, err := GetCloudProvider(i)
	if err != nil {
		return err
	}

	oldData, err := util.GetDataFormFile(i.GetName())
	if err != nil {
		return err
	}
	log.Info("Merging cloud data...")
	res, err := MergeCloudProvider(oldData, curData)
	if err != nil {
		return err
	}

	//err = WriteCloudProvider(oldData,"cloud_old.json")
	//if err!=nil {
	//	return err
	//}
	log.Info("Writing cloud data...")
	err = WriteCloudProvider(res)
	if err != nil {
		return err
	}
	return nil
}

//If kubeData.version exists in old data, then
// 		if kubeData.Envs is empty, then delete it,
//      otherwise, replace it
//If kubeData.version doesn't exists in old data, then append it
func MergeKubernetesSupport(data *v1.CloudProvider, kubeData *v1.KubernetesVersion) (*v1.CloudProvider, error) {
	foundIndex := -1
	for index, k := range data.Spec.KubernetesVersions {
		if version.CompareKubeAwareVersionStrings(k.Spec.GitVersion, kubeData.Spec.GitVersion) == 0 {
			foundIndex = index
		}
	}
	if foundIndex == -1 { //append
		data.Spec.KubernetesVersions = append(data.Spec.KubernetesVersions, *kubeData)
	} else { //replace
		data.Spec.KubernetesVersions[foundIndex] = *kubeData
	}
	return data, nil
}

func AddKubernetesSupport(opts *options.KubernetesData) error {
	kubeData := &v1.KubernetesVersion{}
	kubeData.Spec.GitVersion = opts.Version
	kubeData.Spec.Envs = map[string]bool{}
	for _, env := range opts.Envs {
		if len(env) > 0 {
			kubeData.Spec.Envs[env] = opts.Deprecated
		}
	}
	for _, name := range providers {
		if opts.Provider != options.AllProvider && opts.Provider != name {
			continue
		}
		log.Infof("Getting cloud data for `%v` provider", name)
		data, err := util.GetDataFormFile(name)
		if err != nil {
			return err
		}
		log.Infof("Adding Kubernetes support for `%v` provider", name)
		data, err = MergeKubernetesSupport(data, kubeData)
		if err != nil {
			return err
		}
		log.Infof("Writing cloud data for `%v` provider", name)
		err = WriteCloudProvider(data)
		if err != nil {
			return err
		}
	}
	return nil
}
