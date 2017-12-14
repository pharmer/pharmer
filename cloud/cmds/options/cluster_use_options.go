package options

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterUseConfig struct {
	ClusterName string
	Overwrite   bool
}

func NewClusterUseConfig() *ClusterUseConfig {
	return &ClusterUseConfig{
		ClusterName: "",
		Overwrite:   true,
	}
}

func (c *ClusterUseConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Overwrite, "overwrite", c.Overwrite, "Overwrite context if found.")
}

func (c *ClusterUseConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		errors.New("missing cluster name")
	}
	if len(args) > 1 {
		errors.New("multiple cluster name provided")
	}
	c.ClusterName = args[0]
	return nil
}
