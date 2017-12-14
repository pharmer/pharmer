package options

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CredentialEditConfig struct {
	File        string
	DoNotDelete bool
	Output      string
}

func NewCredentialEditConfig() *CredentialEditConfig {
	return &CredentialEditConfig{
		File:        "",
		DoNotDelete: false,
		Output:      "yaml",
	}
}

func (c *CredentialEditConfig) AddCredentialEditFlags(fs *pflag.FlagSet) {
	fs.StringP("file", "f", "", "Load credential data from file")
	fs.BoolP("do-not-delete", "", false, "Set do not delete flag")
	fs.StringP("output", "o", "yaml", "Output format. One of: yaml|json.")
}

func (c *CredentialEditConfig) ValidateCredentialEditFlags(cmd *cobra.Command, args []string) error {

	if len(args) == 0 {
		return errors.New("Missing credential name")
	}
	if len(args) > 1 {
		return errors.New("Multiple credential name provided.")
	}
	return nil
}
