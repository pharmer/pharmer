package providers

import (
	"fmt"
	"os"

	"github.com/pharmer/cloud/pkg/apis"
	"github.com/pharmer/cloud/pkg/providers/aws"
	"github.com/pharmer/cloud/pkg/providers/azure"
	"github.com/pharmer/cloud/pkg/providers/digitalocean"
	"github.com/pharmer/cloud/pkg/providers/gce"
	"github.com/pharmer/cloud/pkg/providers/linode"
	"github.com/pharmer/cloud/pkg/providers/packet"
	"github.com/pharmer/cloud/pkg/providers/scaleway"
	"github.com/pharmer/cloud/pkg/providers/vultr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	Provider string
	Do       digitalocean.Options
	Packet   packet.Options
	GCE      gce.Options
	AWS      aws.Options
	Azure    azure.Options
	Vultr    vultr.Options
	Linode   linode.Options
	Scaleway scaleway.Options
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
	switch c.Provider {
	case apis.GCE:
		c.GCE.Validate(cmd)
		break
	case apis.DigitalOcean:
		c.Do.Validate(cmd)
		break
	case apis.Packet:
		c.Packet.Validate(cmd)
		break
	case apis.AWS:
		c.AWS.Validate(cmd)
	case apis.Azure:
		c.Azure.Validate(cmd)
		break
	case apis.Vultr:
		c.Vultr.Validate(cmd)
		break
	case apis.Linode:
		c.Linode.Validate(cmd)
		break
	case apis.Scaleway:
		c.Scaleway.Validate(cmd)
		break
	default:
		fmt.Println("missing --provider flag")
		os.Exit(1)
		break
	}
	return nil
}
