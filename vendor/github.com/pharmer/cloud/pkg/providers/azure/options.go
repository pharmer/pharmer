package azure

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.TenantID, "azure.tenant_id", c.TenantID, "provide this flag when provider is azure")
	fs.StringVar(&c.SubscriptionID, "azure.subscription_id", c.SubscriptionID, "provide this flag when provider is azure")
	fs.StringVar(&c.ClientID, "azure.client_id", c.ClientID, "provide this flag when provider is azure")
	fs.StringVar(&c.ClientSecret, "azure.client_secret", c.ClientSecret, "provide this flag when provider is azure")
}

func (c *Options) Validate(cmd *cobra.Command) error {
	flags.EnsureRequiredFlags(cmd, "azure.tenant_id", "azure.subscription_id", "azure.client_id", "azure.client_secret")
	return nil
}
