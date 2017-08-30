package data

import (
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/api"
)

const (
	KeyClusterCredential = "pharmer.appscode.com/cluster-credential"
	KeyDNSCredential     = "pharmer.appscode.com/dns-credential"
	KeyStorageCredential = "pharmer.appscode.com/storage-credential"
)

type CloudData struct {
	Name               string              `json:"name"`
	Env                []string            `json:"env"`
	Regions            []Region            `json:"regions"`
	InstanceTypes      []InstanceType      `json:"instanceTypes"`
	Credentials        []CredentialFormat  `json:"credentials"`
	KubernetesVersions []KubernetesVersion `json:"kubernetesVersions"`
}

func (cd CloudData) Available(env _env.Environment) bool {
	for _, e := range cd.Env {
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
	Version     string           `json:"version"`
	Description string           `json:"description"`
	DefaultSpec *api.ClusterSpec `json:"defaultSpec"`
	Env         map[string]bool  `json:"env,omitempty"`
}

func (v KubernetesVersion) Released(env _env.Environment) bool {
	_, found := v.Env[env.String()]
	return found
}

func (v KubernetesVersion) Deprecated(env _env.Environment) bool {
	deprecated, found := v.Env[env.String()]
	return found && deprecated
}
