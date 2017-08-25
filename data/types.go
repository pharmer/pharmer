package data

import (
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/api"
)

type CloudData struct {
	Name          string             `json:"name"`
	Env           []string           `json:"env"`
	Regions       []Region           `json:"regions"`
	InstanceTypes []InstanceType     `json:"instanceTypes"`
	Credentials   []CredentialFormat `json:"credentials"`
	Kubernetes    ClusterProvider    `json:"kubernetes"`
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
	Type          string `json:"type"`
	DisplayFormat string `json:"displayFormat"`
	Fields        []struct {
		JSON  string `json:"json"`
		Name  string `json:"name"`
		Label string `json:"label"`
		Input string `json:"input"`
	} `json:"fields"`
}

type ClusterProvider struct {
	DefaultSpec *api.ClusterSpec `json:"defaultSpec"`
	Versions    []ClusterVersion `json:"versions"`
}

type ClusterVersion struct {
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Tools       map[string]string `json:"tools"`
	Env         map[string]bool   `json:"env,omitempty"`
}

func (v ClusterVersion) Available(env _env.Environment) bool {
	_, found := v.Env[env.String()]
	return found
}

func (v ClusterVersion) Deprecated(env _env.Environment) bool {
	deprecated, found := v.Env[env.String()]
	return found && deprecated
}
