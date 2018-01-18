package azure

import (
	"github.com/appscode/go/log"
	"github.com/hashicorp/go-version"
	"github.com/pharmer/pharmer/data"
	"github.com/pharmer/pharmer/hack/gendata/util"
)

func GetDefault(versions string) (*AzureDefaultData, error) {
	d := &AzureDefaultData{
		Name: "azure",
		Envs: []string{
			"dev",
			"qa",
			"prod",
		},
		Credentials: []data.CredentialFormat{
			{
				Provider:      "Azure",
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
						Envconfig: "AZURE_TENANT_ID",
						Form:      "azure_tenant_id",
						JSON:      "tenantID",
						Label:     "Tenant Id",
						Input:     "text",
					},
					{
						Envconfig: "AZURE_SUBSCRIPTION_ID",
						Form:      "azure_subscription_id",
						JSON:      "subscriptionID",
						Label:     "Subscription Id",
						Input:     "text",
					},
					{
						Envconfig: "AZURE_CLIENT_ID",
						Form:      "azure_client_id",
						JSON:      "clientID",
						Label:     "Client Id",
						Input:     "text",
					},
					{
						Envconfig: "AZURE_CLIENT_SECRET",
						Form:      "azure_client_secret",
						JSON:      "clientSecret",
						Label:     "Client Secret",
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
