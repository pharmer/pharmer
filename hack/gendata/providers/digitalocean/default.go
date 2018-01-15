package digitalocean

import (
	"github.com/appscode/go/log"
	"github.com/hashicorp/go-version"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

func GetDefault(versions string) (*DigitalOceanDefaultData, error) {
	d := &DigitalOceanDefaultData{
		Name: "digitalocean",
		Envs: []string{
			"dev",
			"qa",
			"prod",
		},
		Credentials: []data.CredentialFormat{
			{
				Provider:      "DigitalOcean",
				DisplayFormat: "field",
				Annotations: map[string]string{
					"pharmer.appscode.com/cluster-credential": "",
					"pharmer.appscode.com/dns-credential":     "",
				},
				Fields: []struct {
					Envconfig string `json:"envconfig"`
					Form      string `json:"form"`
					JSON      string `json:"json"`
					Label     string `json:"label"`
					Input     string `json:"input"`
				}{
					{
						Envconfig: "DIGITALOCEAN_TOKEN",
						Form:      "digitalocean_token",
						JSON:      "token",
						Label:     "Personal Access Token",
						Input:     "password",
					},
				},
			},
		},
		Kubernetes: []data.Kubernetes{},
	}
	vers := util.ParseVersions(versions)
	log.Debug(vers)
	// adding supported kubernetes versions
	for _, v := range vers {
		ver, err := version.NewVersion(v)
		if err != nil {
			return nil, err
		}
		d.Kubernetes = append(d.Kubernetes, data.Kubernetes{
			Version: ver,
			Envs: map[string]bool{
				"dev":  true,
				"qa":   true,
				"prod": true,
			},
		})
	}
	return d, nil
}
