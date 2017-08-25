package files

import (
	"encoding/json"
	"fmt"

	_env "github.com/appscode/go/env"
	"github.com/appscode/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/data"
	"github.com/appscode/pharmer/data/files/aws"
	"github.com/appscode/pharmer/data/files/azure"
	"github.com/appscode/pharmer/data/files/digitalocean"
	"github.com/appscode/pharmer/data/files/gce"
	"github.com/appscode/pharmer/data/files/linode"
	"github.com/appscode/pharmer/data/files/packet"
	"github.com/appscode/pharmer/data/files/scaleway"
	"github.com/appscode/pharmer/data/files/softlayer"
	"github.com/appscode/pharmer/data/files/vultr"
)

type cloudData struct {
	Name          string
	Regions       map[string]data.Region
	InstanceTypes map[string]data.InstanceType
	DefaultSpec   *api.ClusterSpec
	Versions      map[string]data.ClusterVersion
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
		DefaultSpec:   cd.Kubernetes.DefaultSpec,
		Regions:       map[string]data.Region{},
		InstanceTypes: map[string]data.InstanceType{},
		Versions:      map[string]data.ClusterVersion{},
	}
	for _, r := range cd.Regions {
		cloud.Regions[r.Region] = r
	}
	for _, t := range cd.InstanceTypes {
		cloud.InstanceTypes[t.SKU] = t
	}
	for _, v := range cd.Kubernetes.Versions {
		if v.Available(env) {
			cloud.Versions[v.Version] = v
		}
	}
	if cloud.DefaultSpec != nil {
		for _, ng := range cloud.DefaultSpec.NodeGroups {
			if _, found := cloud.InstanceTypes[ng.SKU]; !found {
				return fmt.Errorf("Invalid instance type %s for cloud provider %s", ng.SKU, cloud.Name)
			}
		}
	}
	if len(cloud.Versions) > 0 {
		if _, exists := clouds[cloud.Name]; exists {
			return fmt.Errorf("Redeclared cloud provider %s", cloud.Name)
		}
		clouds[cloud.Name] = cloud
	}

	for _, c := range cd.Credentials {
		if _, exists := credentials[c.Type]; exists {
			return fmt.Errorf("Redeclared credential type %s in cloud provider %s", c.Type, cloud.Name)
		}
		credentials[c.Type] = c
	}

	return nil
}

func Load(env _env.Environment) error {
	dataFiles := [][]byte{}

	if bytes, err := aws.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := azure.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := digitalocean.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := gce.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := linode.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := packet.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := scaleway.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := softlayer.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	if bytes, err := vultr.Asset("cloud.json"); err != nil {
		return err
	} else {
		dataFiles = append(dataFiles, bytes)
	}

	for _, bytes := range dataFiles {
		if err := parseData(bytes, env); err != nil {
			return err
		}
	}
	return nil
}

func CloudProviders() []string {
	result := make([]string, 0, len(clouds))
	for k := range clouds {
		result = append(result, k)
	}
	return result
}

func GetClusterVersion(provider, version string, env _env.Environment) (*data.ClusterVersion, error) {
	p, found := clouds[provider]
	if !found {
		return nil, fmt.Errorf("Can't find cluster provider %v", provider)
	}
	for _, v := range p.Versions {
		if v.Version == version && v.Available(env) {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("Can't find Kubernetes version %v for %v in %v", version, provider, env)
}

func GetInstanceType(provider, sku string) (*data.InstanceType, error) {
	p, found := clouds[provider]
	if !found {
		return nil, fmt.Errorf("Can't find cluster provider %v", provider)
	}
	s, found := p.InstanceTypes[sku]
	if !found {
		return nil, fmt.Errorf("Can't find instance type %s for provider %s.", sku, provider)
	}
	return &s, nil
}

func CredentialTypes() []string {
	result := make([]string, 0, len(credentials))
	for k := range credentials {
		result = append(result, k)
	}
	return result
}

func GetCredetialFormat(ct string) (data.CredentialFormat, bool) {
	cf, found := credentials[ct]
	return cf, found
}
