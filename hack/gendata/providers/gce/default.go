package gce

import (
	"github.com/pharmer/pharmer/data"
	"github.com/hashicorp/go-version"
	"github.com/appscode/go/log"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

func GetDefault(versions string) (*GceDefaultData, error) {
  d := &GceDefaultData{
		Name: "gce",
		Envs: []string{
			"dev",
			"qa",
			"prod",
		},
		Credentials: []data.CredentialFormat{
			{
				Provider: "GoogleCloud",
				DisplayFormat: "json",
				Annotations: map[string]string{
					"pharmer.appscode.com/cluster-credential": "",
					"pharmer.appscode.com/dns-credential": "",
					"pharmer.appscode.com/storage-credential": "",
				},
				Fields: []struct {
					Envconfig string `json:"envconfig"`
					Form      string `json:"form"`
					JSON      string `json:"json"`
					Label     string `json:"label"`
					Input     string `json:"input"`
				}{
					{
						Envconfig: "GCE_PROJECT_ID",
						Form: "gce_project_id",
						JSON: "projectID",
						Label: "Google Cloud Project ID",
						Input: "text",
					},
					{
						Envconfig: "GCE_SERVICE_ACCOUNT",
						Form: "gce_service_account",
						JSON: "serviceAccount",
						Label: "Google Cloud Service Account",
						Input: "textarea",
					},
				},

			},
		},
		Kubernetes: []data.Kubernetes{},
	}
	vers := util.ParseVersions(versions)
	log.Debug(vers)
	// adding supported kubernetes versions
	for _,v := range vers {
		ver, err := version.NewVersion(v)
		if err!= nil {
			return nil, err
		}
		d.Kubernetes = append(d.Kubernetes,data.Kubernetes{
			Version:ver,
			Envs: map[string]bool {
				"dev": true,
				"qa": true,
				"prod": true,
			},
		})
	}
	return d,nil
}