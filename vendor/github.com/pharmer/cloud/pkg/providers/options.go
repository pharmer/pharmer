package providers

import (
	"fmt"
	"os"

	"github.com/appscode/go/flags"
	"github.com/pharmer/cloud/pkg/apis"
	v1 "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
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

func NewOptionsForCredential(c v1.Credential) Options {
	opt := Options{
		Provider: c.Spec.Provider,
	}
	var commonSpec credential.CommonSpec
	commonSpec.Provider = c.Spec.Provider
	commonSpec.Data = c.Spec.Data

	switch c.Spec.Provider {
	case apis.AWS:
		opt.AWS = credential.AWS{CommonSpec: commonSpec}
	case apis.GCE:
		opt.GCE = credential.GCE{CommonSpec: commonSpec}
	case apis.DigitalOcean:
		opt.Do = credential.DigitalOcean{CommonSpec: commonSpec}
	case apis.Packet:
		opt.Packet = credential.Packet{CommonSpec: commonSpec}
	case apis.Azure:
		opt.Azure = credential.Azure{CommonSpec: commonSpec}
	case apis.Vultr:
		opt.Vultr = credential.Vultr{CommonSpec: commonSpec}
	case apis.Linode:
		opt.Linode = credential.Linode{CommonSpec: commonSpec}
	case apis.Scaleway:
		opt.Scaleway = credential.Scaleway{CommonSpec: commonSpec}
	default:
		panic("unknown provider " + c.Spec.Provider)
	}
	return opt
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
