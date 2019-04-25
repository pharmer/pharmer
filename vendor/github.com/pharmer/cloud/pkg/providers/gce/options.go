package gce

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Options struct {
	CredentialFile string
	ProjectID      string
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.CredentialFile, "gce.credential_file", "c", c.CredentialFile, "Location of cloud credential file (required when --provider=gce)")
	fs.StringVar(&c.ProjectID, "gce.project_id", c.ProjectID, "provide this flag when provider is gce")
}

func (c *Options) Validate(cmd *cobra.Command) error {
	flags.EnsureRequiredFlags(cmd, "gce.credential_file", "gce.project_id")
	return nil
}
