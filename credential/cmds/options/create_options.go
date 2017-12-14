package options

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialCreateConfig struct {
	Name     string
	Provider string
	FromEnv  bool
	FromFile string
	Issue    bool
}

func NewCredentialCreateConfig() *CredentialCreateConfig {
	return &CredentialCreateConfig{
		Provider: "",
		FromEnv:  false,
		FromFile: "",
		Issue:    false,
	}
}

func (c *CredentialCreateConfig) AddCredentialCreateFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider")
	fs.BoolVarP(&c.FromEnv, "from-env", "l", c.FromEnv, "Load credential data from ENV.")
	fs.StringVarP(&c.FromFile, "from-file", "f", c.FromFile, "Load credential data from file")
	fs.BoolVar(&c.Issue, "issue", c.Issue, "Issue credential")
}

func (c *CredentialCreateConfig) ValidateCredentialCreateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		errors.New("Missing credential name.")
	}
	if len(args) > 1 {
		errors.New("Multiple credential name provided.")
	}
	c.Name = args[0]
	return nil
}
