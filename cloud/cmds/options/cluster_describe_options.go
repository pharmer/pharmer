package options

import (
	"strings"

	"github.com/pharmer/pharmer/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterDescribeConfig struct {
	Clusters []string
	Owner    string
}

func NewClusterDescribeConfig() *ClusterDescribeConfig {
	return &ClusterDescribeConfig{
		Owner: utils.GetLocalOwner(),
	}
}

func (c *ClusterDescribeConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")
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
