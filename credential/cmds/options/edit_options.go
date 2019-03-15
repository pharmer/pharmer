package options

import (
	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialEditConfig struct {
	Name        string
	File        string
	DoNotDelete bool
	Output      string
	Owner       string
}

func NewCredentialEditConfig() *CredentialEditConfig {
	return &CredentialEditConfig{
		File:        "",
		DoNotDelete: false,
		Output:      "yaml",
		Owner:       utils.GetLocalOwner(),
	}
}

func (c *CredentialEditConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringP("file", "f", "", "Load credential data from file")
	fs.BoolP("do-not-delete", "", false, "Set do not delete flag")
	fs.StringP("output", "o", "yaml", "Output format. One of: yaml|json.")
	fs.StringVarP(&c.Owner, "owner", "", c.Owner, "Current user id")
}

func (c *CredentialEditConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing credential name")
	}
	if len(args) > 1 {
		return errors.New("multiple credential name provided")
	}
	c.Name = args[0]
	return nil
}
