package aws

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Region, "aws.region", c.Region, "provide this flag when provider is aws")
	fs.StringVar(&c.AccessKeyID, "aws.access_key_id", c.AccessKeyID, "provide this flag when provider is aws")
	fs.StringVar(&c.SecretAccessKey, "aws.secret_access_key", c.SecretAccessKey, "provide this flag when provider is aws")
}

func (c *Options) Validate(cmd *cobra.Command) error {
	flags.EnsureRequiredFlags(cmd, "aws.region", "aws.access_key_id", "aws.secret_access_key")
	return nil
}
