package files

import (
	"encoding/json"
	"sort"

	_env "github.com/appscode/go/env"
	"github.com/appscode/go/log"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/data/files/aws"
	"github.com/pharmer/pharmer/data/files/azure"
	"github.com/pharmer/pharmer/data/files/digitalocean"
	"github.com/pharmer/pharmer/data/files/gce"
	"github.com/pharmer/pharmer/data/files/linode"
	"github.com/pharmer/pharmer/data/files/ovh"
	"github.com/pharmer/pharmer/data/files/packet"
	"github.com/pharmer/pharmer/data/files/scaleway"
	"github.com/pharmer/pharmer/data/files/softlayer"
	"github.com/pharmer/pharmer/data/files/vultr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

type cloudData struct {
	Name          string
	Regions       map[string]data.Region
	InstanceTypes map[string]data.InstanceType
	Versions      []data.Kubernetes
}

var (
	clouds      = map[string]cloudData{}
	credentials = map[string]data.CredentialFormat{}
)

func parseData(bytes []byte, env _env.Environment) error {
	var cd data.CloudData
	err := json.Unmarshal(bytes, &cd)
	if err != nil {
		return err
	}
	if !cd.Available(env) {
		log.Infof("Skipping loading cloud provider %s, as not enabled in environment %s", cd.Name, env)
		return nil
	}

	cloud := cloudData{
		Name:          cd.Name,
		Regions:       map[string]data.Region{},
		InstanceTypes: map[string]data.InstanceType{},
		Versions:      make([]data.Kubernetes, 0),
	}
	for _, r := range cd.Regions {
		cloud.Regions[r.Region] = r
	}
	for _, t := range cd.InstanceTypes {
		cloud.InstanceTypes[t.SKU] = t
	}

	kubes := make([]data.Kubernetes, 0, len(cd.Kubernetes))
	for _, v := range cd.Kubernetes {
		if v.Released(env) {
			kubes = append(kubes, v)
		}
	}
	sort.Slice(kubes, func(i, j int) bool { return kubes[i].Version.LessThan(kubes[j].Version) })
	cloud.Versions = kubes

	if len(cloud.Versions) > 0 {
		if _, exists := clouds[cloud.Name]; exists {
			return errors.Errorf("redeclared cloud provider %s", cloud.Name)
		}
		clouds[cloud.Name] = cloud
	}

	for _, c := range cd.Credentials {
		if _, exists := credentials[c.Provider]; exists {
			return errors.Errorf("redeclared credential type %s in cloud provider %s", c.Provider, cloud.Name)
		}
		credentials[c.Provider] = c
	}

	return nil
}

func LoadDataFiles() ([][]byte, error) {
	dataFiles := [][]byte{}

	if bytes, err := aws.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := azure.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := digitalocean.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := gce.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := linode.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := packet.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := scaleway.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := softlayer.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := vultr.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := ovh.Asset("cloud.json"); err != nil {
		return nil, err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	return dataFiles, nil
}

func Load(env _env.Environment) error {
	dataFiles, err := LoadDataFiles()
	if err != nil {
		return err
	}
	for _, bytes := range dataFiles {
		if err := parseData(bytes, env); err != nil {
			return err
		}
	}
	return nil
}

func GetInstanceType(provider, sku string) (*data.InstanceType, error) {
	p, found := clouds[provider]
	if !found {
		return nil, errors.Errorf("can't find cluster provider %v", provider)
	}
	s, found := p.InstanceTypes[sku]
	if !found {
		return nil, errors.Errorf("can't find instance type %s for provider %s", sku, provider)
	}
	return &s, nil
}

func CredentialProviders() sets.String {
	result := sets.String{}
	for k := range credentials {
		result.Insert(k)
	}
	return result
}

func ClusterProviders() sets.String {
	result := sets.String{}
	for k, v := range credentials {
		if _, ok := v.Annotations[data.KeyClusterCredential]; ok {
			result.Insert(k)
		}
	}
	return result
}

func DNSProviders() sets.String {
	result := sets.String{}
	for k, v := range credentials {
		if _, ok := v.Annotations[data.KeyDNSCredential]; ok {
			result.Insert(k)
		}
	}
	return result
}

func StorageProviders() sets.String {
	result := sets.String{}
	for k, v := range credentials {
		if _, ok := v.Annotations[data.KeyStorageCredential]; ok {
			result.Insert(k)
		}
	}
	return result
}

func GetCredentialFormat(provider string) (data.CredentialFormat, bool) {
	cf, found := credentials[provider]
	return cf, found
}

func GetRegions(provider string) (map[string]data.Region, error) {
	p, found := clouds[provider]
	if !found {
		return nil, errors.Errorf("can't find cluster provider %v", provider)
	}
	return p.Regions, nil
}

func GetRegionsFromZone(provider, zone string) (data.Region, error) {
	regions, err := GetRegions(provider)
	if err != nil {
		return data.Region{}, err
	}
	for _, region := range regions {
		zones := sets.NewString(region.Zones...)
		if zones.Has(zone) {
			return region, nil
		}
	}
	return data.Region{}, errors.Errorf("can't find zone %v for provider %v", zone, provider)
}

func GetInstanceByRegionCPU(provider, region string, cpu int) (*data.InstanceType, error) {
	p, found := clouds[provider]
	if !found {
		return nil, errors.Errorf("can't find cluster provider %v", provider)
	}

	for _, instance := range p.InstanceTypes {
		regions := sets.NewString(instance.Regions...)
		if regions.Has(region) && instance.CPU >= cpu {
			return &instance, nil
		}
	}
	return nil, errors.Errorf("can't find instance for provider %v with region %v and cpu %v", provider, region, cpu)
}

func GetInstanceByZoneCPU(provider, zone string, cpu int) (*data.InstanceType, error) {
	p, found := clouds[provider]
	if !found {
		return nil, errors.Errorf("can't find cluster provider %v", provider)
	}

	for _, instance := range p.InstanceTypes {
		zones := sets.NewString(instance.Zones...)
		if zones.Has(zone) && instance.CPU >= cpu {
			return &instance, nil
		}
	}
	return nil, errors.Errorf("can't find instance for provider %v with zone %v and cpu %v", provider, zone, cpu)

}
