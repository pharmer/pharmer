package options

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialDeleteConfig struct {
	Credentials []string
}

func NewCredentialDeleteConfig() *CredentialDeleteConfig {
	return &CredentialDeleteConfig{}
}

func (c *CredentialDeleteConfig) AddCredentialDeleteFlags(fs *pflag.FlagSet) {
}

func (c *CredentialDeleteConfig) ValidateCredentialDeleteFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		errors.New("Missing Credential name.")
	}
	c.Credentials = args
	return nil
}
