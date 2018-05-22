package options

import (
	"strings"

	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeGroupNodeDeleteConfig struct {
	ClusterName string
}

func NewNodeGroupDeleteConfig() *NodeGroupNodeDeleteConfig {
	return &NodeGroupNodeDeleteConfig{
		ClusterName: "",
	}
}

func (c *NodeGroupNodeDeleteConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of the Kubernetes cluster")
}

func (c *NodeGroupNodeDeleteConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	flags.EnsureRequiredFlags(cmd, "cluster")
	c.ClusterName = strings.ToLower(c.ClusterName)
	return nil
}
