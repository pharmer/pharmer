package options

import (
	"strings"

	"github.com/spf13/cobra"
)

type ClusterDescribeConfig struct {
	Clusters []string
}

func NewClusterDescribeConfig() *ClusterDescribeConfig {
	return &ClusterDescribeConfig{}
}

func (c *ClusterDescribeConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	c.Clusters = func(names []string) []string {
		for i := range names {
			names[i] = strings.ToLower(names[i])
		}
		return names
	}(args)
	return nil
}
