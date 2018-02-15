package options

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialDeleteConfig struct {
	Credentials []string
}

func NewCredentialDeleteConfig() *CredentialDeleteConfig {
	return &CredentialDeleteConfig{}
}

func (c *CredentialDeleteConfig) AddFlags(fs *pflag.FlagSet) {
}

func (c *CredentialDeleteConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		errors.New("missing Credential name")
	}
	c.Credentials = args
	return nil
}
