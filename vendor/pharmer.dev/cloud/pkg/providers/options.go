/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package providers

import (
	"fmt"
	"os"

	"pharmer.dev/cloud/apis"
	v1 "pharmer.dev/cloud/apis/cloud/v1"
	"pharmer.dev/cloud/pkg/credential"

	"github.com/appscode/go/flags"
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
	case apis.DigitalOcean:
		required = c.Do.RequiredFlags()
	case apis.Packet:
		required = c.Packet.RequiredFlags()
	case apis.AWS:
		required = c.AWS.RequiredFlags()
	case apis.Azure:
		required = c.Azure.RequiredFlags()
	case apis.Vultr:
		required = c.Vultr.RequiredFlags()
	case apis.Linode:
		required = c.Linode.RequiredFlags()
	case apis.Scaleway:
		required = c.Scaleway.RequiredFlags()
	default:
		fmt.Println("missing --provider flag")
		os.Exit(1)
	}

	if len(required) > 0 {
		flags.EnsureRequiredFlags(cmd, required...)
	}
	return nil
}
