package options

import (
	"github.com/spf13/pflag"
	"github.com/spf13/cobra"
	"github.com/appscode/go/flags"
)

const (
	DefaultKubernetesVersion string = "1.8.0"
)

type CloudData struct {
	Provider       string
	//credential file for gce
	CredentialFile string
	//access token for digitalocean
	DoToken        string
	//access token for packet
	PacketToken string
	GCEProjectName string
	AWSRegion string
	AWSAccessKeyID string
	AWSSecretAccessKey string
	KubernetesVersions string
	AzureTenantId string
	AzureSubscriptionId string
	AzureClientId string
	AzureClientSecret string
}



func NewCloudData() *CloudData {
	return &CloudData{
		Provider: "",
		KubernetesVersions: DefaultKubernetesVersion,
	}
}

func (c *CloudData) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider")
	fs.StringVarP(&c.CredentialFile, "credential-file", "c", c.CredentialFile, "Location of cloud credential file (required when --provider=gce)")
	fs.StringVar(&c.GCEProjectName, "google-project", c.GCEProjectName, "When using the Google provider, specify the Google project (required when --provider=gce)")
	fs.StringVar(&c.KubernetesVersions, "versions-support", c.KubernetesVersions, "Supported versions of kubernetes, example: --versions-support=1.1.0,1.2.0")
	fs.StringVar(&c.DoToken, "do-token", c.DoToken, "provide this flag when provider is digitalocean")
	fs.StringVar(&c.PacketToken, "packet-token", c.PacketToken, "provide this flag when provider is packet")
	fs.StringVar(&c.AWSRegion, "aws-region", c.AWSRegion, "provide this flag when provider is aws")
	fs.StringVar(&c.AWSAccessKeyID, "aws-access-key-id", c.AWSAccessKeyID, "provide this flag when provider is aws")
	fs.StringVar(&c.AWSSecretAccessKey, "aws-secret-access-key", c.AWSSecretAccessKey, "provide this flag when provider is aws")
	fs.StringVar(&c.AzureTenantId, "azure-tenant-id", c.AzureTenantId, "provide this flag when provider is azure")
	fs.StringVar(&c.AzureSubscriptionId, "azure-subscription-id", c.AzureSubscriptionId, "provide this flag when provider is azure")
	fs.StringVar(&c.AzureClientId, "azure-client-id", c.AzureClientId, "provide this flag when provider is azure")
	fs.StringVar(&c.AzureClientSecret, "azure-client-secret", c.AzureClientSecret, "provide this flag when provider is azure")
}

func (c *CloudData) ValidateFlags(cmd *cobra.Command, args []string) error {
	var ensureFlags []string
	switch c.Provider {
	case "gce":
		ensureFlags = []string{"provider",  "credential-file", "google-project"}
		break
	case "digitalocean":
		ensureFlags = []string{"provider",  "do-token"}
		break
	case "packet":
		ensureFlags = []string{"provider",  "packet-token"}
		break
	case "aws":
		ensureFlags = []string{"provider",  "aws-region","aws-access-key-id","aws-secret-access-key"}
	case "azure":
		ensureFlags = []string{"provider",  "azure-tenant-id", "azure-subscription-id", "azure-client-id", "azure-client-secret"}
		break
	default:
		ensureFlags = []string{"provider"}
		break
	}

	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	return nil
}
