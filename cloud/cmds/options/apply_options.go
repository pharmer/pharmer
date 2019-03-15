package options

import (
	"strings"

	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ApplyConfig struct {
	ClusterName string
	DryRun      bool
	Owner       string
}

func NewApplyConfig() *ApplyConfig {
	return &ApplyConfig{
		ClusterName: "",
		DryRun:      false,
		Owner:       utils.GetLocalOwner(),
	}
}

func (c *ApplyConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.DryRun, "dry-run", c.DryRun, "Dry run.")
	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")

}

func (c *ApplyConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing cluster name")
	}
	if len(args) > 1 {
		return errors.New("multiple cluster name provided.")
	}
	c.ClusterName = strings.ToLower(args[0])
	return nil
}
