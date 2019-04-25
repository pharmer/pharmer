package digitalocean

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	Token string `json:"token"`
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Token, "digitalocean.token", c.Token, "provide this flag when provider is digitalocean")
}

func (c *Options) Validate(cmd *cobra.Command) error {
	flags.EnsureRequiredFlags(cmd, "digitalocean.token")
	return nil
}
