package scaleway

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	Token        string `json:"token"`
	Organization string `json:"organization"`
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Token, "scaleway.token", c.Token, "provide this flag when provider is scaleway")
	fs.StringVar(&c.Organization, "scaleway.organization", c.Organization, "provide this flag when provider is scaleway")
}

func (c *Options) Validate(cmd *cobra.Command) error {
	flags.EnsureRequiredFlags(cmd, "scaleway.token", "scaleway.organization")
	return nil
}
