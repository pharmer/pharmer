package aws

import (
	"github.com/appscode/go/log"
	"github.com/hashicorp/go-version"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

func GetDefault(versions string) (*AwsDefaultData, error) {
	d := &AwsDefaultData{
		Name: "aws",
		Envs: []string{
			"dev",
			"qa",
			"prod",
		},
		Credentials: []data.CredentialFormat{
			{
				Provider:      "AWS",
				DisplayFormat: "field",
				Annotations: map[string]string{
					"pharmer.appscode.com/cluster-credential": "",
					"pharmer.appscode.com/dns-credential":     "",
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
						Envconfig: "AWS_ACCESS_KEY_ID",
						Form:      "aws_access_key_id",
						JSON:      "accessKeyID",
						Label:     "Access Key Id",
						Input:     "text",
					},
					{
						Envconfig: "AWS_SECRET_ACCESS_KEY",
						Form:      "aws_secret_access_key",
						JSON:      "secretAccessKey",
						Label:     "Secret Access Key",
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
