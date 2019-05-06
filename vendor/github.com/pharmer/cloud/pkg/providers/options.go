package providers

import (
	"fmt"
	"os"

	"github.com/appscode/go/flags"
	"github.com/pharmer/cloud/pkg/credential"

	"github.com/pharmer/cloud/pkg/apis"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	Provider string
	Do       credential.DigitalOcean
	Packet   credential.Packet
	GCE      credential.GCE
	AWS      credential.AWS
	Azure    credential.Azure
	Vultr    credential.Vultr
	Linode   credential.Linode
	Scaleway credential.Scaleway
}

func NewOptions() *Options {
	return &Options{
		Provider: "",
	}
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider")
	c.Do.AddFlags(fs)
	c.Packet.AddFlags(fs)
	c.GCE.AddFlags(fs)
	c.AWS.AddFlags(fs)
	c.Azure.AddFlags(fs)
	c.Vultr.AddFlags(fs)
	c.Linode.AddFlags(fs)
	c.Scaleway.AddFlags(fs)
}

func (c *Options) ValidateFlags(cmd *cobra.Command, args []string) error {
	var required []string

	switch c.Provider {
	case apis.GCE:
		required = c.GCE.RequiredFlags()
		break
	case apis.DigitalOcean:
		required = c.Do.RequiredFlags()
		break
	case apis.Packet:
		required = c.Packet.RequiredFlags()
		break
	case apis.AWS:
		required = c.AWS.RequiredFlags()
	case apis.Azure:
		required = c.Azure.RequiredFlags()
		break
	case apis.Vultr:
		required = c.Vultr.RequiredFlags()
		break
	case apis.Linode:
		required = c.Linode.RequiredFlags()
		break
	case apis.Scaleway:
		required = c.Scaleway.RequiredFlags()
		break
	default:
		fmt.Println("missing --provider flag")
		os.Exit(1)
	}

	if len(required) > 0 {
		flags.EnsureRequiredFlags(cmd, required...)
	}
	return nil
}
