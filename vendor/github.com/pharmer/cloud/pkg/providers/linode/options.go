package linode

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	Token string `json:"token"`
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Token, "linode.token", c.Token, "provide this flag when provider is linode")
}

func (c *Options) Validate(cmd *cobra.Command) error {
	flags.EnsureRequiredFlags(cmd, "linode.api_token")
	return nil
}
