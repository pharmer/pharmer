package options

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialGetConfig struct {
	Credentials []string
	Output      string
}

func NewCredentialGetConfig() *CredentialGetConfig {
	return &CredentialGetConfig{
		Output: "",
	}
}

func (c *CredentialGetConfig) AddCredentialGetFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: json|yaml|wide")
}

func (c *CredentialGetConfig) ValidateCredentialGetFlags(cmd *cobra.Command, args []string) error {
	c.Credentials = args
	return nil
}
