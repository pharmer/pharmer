package data

import (
	version "github.com/appscode/go-version"
	_env "github.com/appscode/go/env"
)

const (
	KeyClusterCredential = "pharmer.appscode.com/cluster-credential"
	KeyDNSCredential     = "pharmer.appscode.com/dns-credential"
	KeyStorageCredential = "pharmer.appscode.com/storage-credential"
)

type CloudData struct {
	Name          string             `json:"name"`
	Envs          []string           `json:"envs"`
	Regions       []Region           `json:"regions"`
	InstanceTypes []InstanceType     `json:"instanceTypes"`
	Credentials   []CredentialFormat `json:"credentials"`
	Kubernetes    []Kubernetes       `json:"kubernetes"`
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
	Category    string      `json:"category,omitempty"`
	CPU         int         `json:"cpu"`
	RAM         interface{} `json:"ram"`
	Disk        int         `json:"disk,omitempty"`
	Regions     []string    `json:"regions,omitempty"`
	Zones       []string    `json:"zones,omitempty"`
	Deprecated  bool        `json:"deprecated,omitempty"`
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

type Kubernetes struct {
	Version *version.Version `json:"version"`
	Envs    map[string]bool  `json:"envs,omitempty"`
}

func (v Kubernetes) Released(env _env.Environment) bool {
	_, found := v.Envs[env.String()]
	return found
}

func (v Kubernetes) Deprecated(env _env.Environment) bool {
	deprecated, found := v.Envs[env.String()]
	return found && deprecated
}
