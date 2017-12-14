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

func (c *ClusterDescribeConfig) AddClusterDescribeFlags(fs *pflag.FlagSet) {
}

func (c *ClusterDescribeConfig) ValidateClusterDescribeFlags(cmd *cobra.Command, args []string) error {
	c.Clusters = args
	return nil
}
