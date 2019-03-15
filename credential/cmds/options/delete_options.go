package options

import (
	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialDeleteConfig struct {
	Credentials []string
	Owner       string
}

func NewCredentialDeleteConfig() *CredentialDeleteConfig {
	return &CredentialDeleteConfig{Owner: utils.GetLocalOwner()}
}

func (c *CredentialDeleteConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")
}

func (c *CredentialDeleteConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		errors.New("missing Credential name")
	}
	c.Credentials = args
	return nil
}
