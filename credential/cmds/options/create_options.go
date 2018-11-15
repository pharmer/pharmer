package options

import (
	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialCreateConfig struct {
	Name     string
	Provider string
	FromEnv  bool
	FromFile string
	Issue    bool
	Owner    string
}

func NewCredentialCreateConfig() *CredentialCreateConfig {
	return &CredentialCreateConfig{
		Provider: "",
		FromEnv:  false,
		FromFile: "",
		Issue:    false,
		Owner:    utils.GetLocalOwner(),
	}
}

func (c *CredentialCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider")
	fs.BoolVarP(&c.FromEnv, "from-env", "l", c.FromEnv, "Load credential data from ENV.")
	fs.StringVarP(&c.FromFile, "from-file", "f", c.FromFile, "Load credential data from file")
	fs.BoolVar(&c.Issue, "issue", c.Issue, "Issue credential")

	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")
}

func (c *CredentialCreateConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		errors.New("missing credential name")
	}
	if len(args) > 1 {
		errors.New("multiple credential name provided")
	}
	c.Name = args[0]
	return nil
}
