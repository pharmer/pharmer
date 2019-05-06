package credential

import (
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Azure struct {
	CommonSpec

	tenantID       string
	subscriptionID string
	clientID       string
	clientSecret   string
}

func (c Azure) ClientID() string       { return get(c.Data, AzureClientID, c.clientID) }
func (c Azure) ClientSecret() string   { return get(c.Data, AzureClientSecret, c.clientSecret) }
func (c Azure) SubscriptionID() string { return get(c.Data, AzureSubscriptionID, c.subscriptionID) }
func (c Azure) TenantID() string       { return get(c.Data, AzureTenantID, c.tenantID) }

func (c *Azure) LoadFromEnv() {
	c.CommonSpec.LoadFromEnv(c.Format())
}

func (c Azure) IsValid() (bool, error) {
	return c.CommonSpec.IsValid(c.Format())
}

func (c *Azure) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.tenantID, apis.Azure+"."+AzureTenantID, c.tenantID, "provide this flag when provider is azure")
	fs.StringVar(&c.subscriptionID, apis.Azure+"."+AzureSubscriptionID, c.subscriptionID, "provide this flag when provider is azure")
	fs.StringVar(&c.clientID, apis.Azure+"."+AzureClientID, c.clientID, "provide this flag when provider is azure")
	fs.StringVar(&c.clientSecret, apis.Azure+"."+AzureClientSecret, c.clientSecret, "provide this flag when provider is azure")
}

func (_ Azure) RequiredFlags() []string {
	return []string{
		apis.Azure + "." + AzureTenantID,
		apis.Azure + "." + AzureSubscriptionID,
		apis.Azure + "." + AzureClientID,
		apis.Azure + "." + AzureClientSecret,
	}
}

func (_ Azure) Format() v1.CredentialFormat {
	return v1.CredentialFormat{
		ObjectMeta: metav1.ObjectMeta{
			Name: apis.Azure + "-cred",
			Labels: map[string]string{
				apis.KeyCloudProvider: apis.Azure,
			},
			Annotations: map[string]string{
				apis.KeyClusterCredential: "",
				apis.KeyDNSCredential:     "",
			},
		},
		Spec: v1.CredentialFormatSpec{
			Provider:      apis.Azure,
			DisplayFormat: "field",
			Fields: []v1.CredentialField{
				{
					Envconfig: "AZURE_TENANT_ID",
					Form:      "azure_tenant_id",
					JSON:      AzureTenantID,
					Label:     "Tenant Id",
					Input:     "text",
				},
				{
					Envconfig: "AZURE_SUBSCRIPTION_ID",
					Form:      "azure_subscription_id",
					JSON:      AzureSubscriptionID,
					Label:     "Subscription Id",
					Input:     "text",
				},
				{
					Envconfig: "AZURE_CLIENT_ID",
					Form:      "azure_client_id",
					JSON:      AzureClientID,
					Label:     "Client Id",
					Input:     "text",
				},
				{
					Envconfig: "AZURE_CLIENT_SECRET",
					Form:      "azure_client_secret",
					JSON:      AzureClientSecret,
					Label:     "Client Secret",
					Input:     "password",
				},
			},
		},
	}
}
