package packet

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	Token string `json:"token"`
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Token, "packet.token", c.Token, "provide this flag when provider is packet")
}

func (c *Options) Validate(cmd *cobra.Command) error {
	flags.EnsureRequiredFlags(cmd, "packet.api_key")
	return nil
}
