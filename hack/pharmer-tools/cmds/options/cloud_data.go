package options

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type GenData struct {
	Provider string
	//credential file for gce
	CredentialFile string
	//access token for digitalocean
	DoToken string
	//access token for packet
	PacketApiKey         string
	GCEProjectID         string
	AWSRegion            string
	AWSAccessKeyID       string
	AWSSecretAccessKey   string
	AzureTenantId        string
	AzureSubscriptionId  string
	AzureClientId        string
	AzureClientSecret    string
	VultrApiToken        string
	LinodeApiToken       string
	ScalewayToken        string
	ScalewayOrganization string
}

func NewGenData() *GenData {
	return &GenData{
		Provider: "",
	}
}

func (c *GenData) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider")
	fs.StringVarP(&c.CredentialFile, "gce.credential_file", "c", c.CredentialFile, "Location of cloud credential file (required when --provider=gce)")
	fs.StringVar(&c.GCEProjectID, "gce.project_id", c.GCEProjectID, "provide this flag when provider is gce")
	fs.StringVar(&c.DoToken, "digitalocean.token", c.DoToken, "provide this flag when provider is digitalocean")
	fs.StringVar(&c.PacketApiKey, "packet.api_key", c.PacketApiKey, "provide this flag when provider is packet")
	fs.StringVar(&c.AWSRegion, "aws.region", c.AWSRegion, "provide this flag when provider is aws")
	fs.StringVar(&c.AWSAccessKeyID, "aws.access_key_id", c.AWSAccessKeyID, "provide this flag when provider is aws")
	fs.StringVar(&c.AWSSecretAccessKey, "aws.secret_access_key", c.AWSSecretAccessKey, "provide this flag when provider is aws")
	fs.StringVar(&c.AzureTenantId, "azure.tenant_id", c.AzureTenantId, "provide this flag when provider is azure")
	fs.StringVar(&c.AzureSubscriptionId, "azure.subscription_id", c.AzureSubscriptionId, "provide this flag when provider is azure")
	fs.StringVar(&c.AzureClientId, "azure.client_id", c.AzureClientId, "provide this flag when provider is azure")
	fs.StringVar(&c.AzureClientSecret, "azure.client_secret", c.AzureClientSecret, "provide this flag when provider is azure")
	fs.StringVar(&c.VultrApiToken, "vultr.api_token", c.VultrApiToken, "provide this flag when provider is vultr")
	fs.StringVar(&c.LinodeApiToken, "linode.api_token", c.LinodeApiToken, "provide this flag when provider is linode")
	fs.StringVar(&c.ScalewayToken, "scaleway.token", c.ScalewayToken, "provide this flag when provider is scaleway")
	fs.StringVar(&c.ScalewayOrganization, "scaleway.organization", c.ScalewayOrganization, "provide this flag when provider is scaleway")
}

func (c *GenData) ValidateFlags(cmd *cobra.Command, args []string) error {
	var ensureFlags []string
	switch c.Provider {
	case "gce":
		ensureFlags = []string{"gce.credential_file", "gce.project_id"}
		break
	case "digitalocean":
		ensureFlags = []string{"digitalocean.token"}
		break
	case "packet":
		ensureFlags = []string{"packet.api_key"}
		break
	case "aws":
		ensureFlags = []string{"aws.region", "aws.access_key_id", "aws.secret_access_key"}
	case "azure":
		ensureFlags = []string{"azure.tenant_id", "azure.subscription_id", "azure.client_id", "azure.client_secret"}
		break
	case "vultr":
		ensureFlags = []string{"vultr.api_token"}
		break
	case "linode":
		ensureFlags = []string{"linode.api_token"}
		break
	case "scaleway":
		ensureFlags = []string{"scaleway.token", "scaleway.organization"}
		break
	default:
		ensureFlags = []string{}
		break
	}
	ensureFlags = append(ensureFlags, "provider")

	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	return nil
}
