package options

import (
	"strings"

	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterUseConfig struct {
	ClusterName string
	Overwrite   bool
	Owner       string
}

func NewClusterUseConfig() *ClusterUseConfig {
	return &ClusterUseConfig{
		ClusterName: "",
		Overwrite:   true,
		Owner:       utils.GetLocalOwner(),
	}
}

func (c *ClusterUseConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Overwrite, "overwrite", c.Overwrite, "Overwrite context if found.")
	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")
}

func (c *ClusterUseConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("missing cluster name")
	}
	if len(args) > 1 {
		return errors.New("multiple cluster name provided")
	}
	c.ClusterName = strings.ToLower(args[0])
	return nil
}
