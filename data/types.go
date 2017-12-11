package data

import (
	_env "github.com/appscode/go/env"
	"github.com/hashicorp/go-version"
)

const (
	KeyClusterCredential = "pharmer.appscode.com/cluster-credential"
	KeyDNSCredential     = "pharmer.appscode.com/dns-credential"
	KeyStorageCredential = "pharmer.appscode.com/storage-credential"
)

type CloudData struct {
	Name               string              `json:"name"`
	Envs               []string            `json:"envs"`
	Regions            []Region            `json:"regions"`
	InstanceTypes      []InstanceType      `json:"instanceTypes"`
	Credentials        []CredentialFormat  `json:"credentials"`
	KubernetesVersions []KubernetesVersion `json:"kubernetesVersions"`
}

func (cd CloudData) Available(env _env.Environment) bool {
	for _, e := range cd.Envs {
		if e == env.String() {
			return true
		}
	}
	return false
}

type Region struct {
	Location string   `json:"location"`
	Region   string   `json:"region"`
	Zones    []string `json:"zones,omitempty"`
}

type InstanceType struct {
	SKU         string      `json:"sku"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	CPU         int         `json:"cpu"`
	RAM         interface{} `json:"ram"`
	Disk        int         `json:"disk"`
	Regions     []string    `json:"regions,omitempty"`
}

type CredentialFormat struct {
	Provider      string            `json:"provider"`
	DisplayFormat string            `json:"displayFormat"`
	Annotations   map[string]string `json:"annotations"`
	Fields        []struct {
		Envconfig string `json:"envconfig"`
		Form      string `json:"form"`
		JSON      string `json:"json"`
		Label     string `json:"label"`
		Input     string `json:"input"`
	} `json:"fields"`
}

type KubernetesVersion struct {
	Version *version.Version `json:"version"`
	Envs    map[string]bool  `json:"envs,omitempty"`
}

func (v KubernetesVersion) Released(env _env.Environment) bool {
	_, found := v.Envs[env.String()]
	return found
}

func (v KubernetesVersion) Deprecated(env _env.Environment) bool {
	deprecated, found := v.Envs[env.String()]
	return found && deprecated
}
