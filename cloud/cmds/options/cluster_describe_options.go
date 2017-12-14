package options

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterDescribeConfig struct {
	Clusters []string
}

func NewClusterDescribeConfig() *ClusterDescribeConfig {
	return &ClusterDescribeConfig{}
}

func (c *ClusterDescribeConfig) AddFlags(fs *pflag.FlagSet) {
}

func (c *ClusterDescribeConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	c.Clusters = args
	return nil
}
