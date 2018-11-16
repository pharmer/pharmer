package options

import (
	"github.com/pharmer/pharmer/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialGetConfig struct {
	Credentials []string
	Output      string
	Owner       string
}

func NewCredentialGetConfig() *CredentialGetConfig {
	return &CredentialGetConfig{
		Output: "",
		Owner:  utils.GetLocalOwner(),
	}
}

func (c *CredentialGetConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: json|yaml|wide")
	fs.StringVarP(&c.Owner, "owner", "", c.Owner, "Current user id")
}

func (c *CredentialGetConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	c.Credentials = args
	return nil
}
