package options

import (
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

func (c *NodeGroupNodeDeleteConfig) AddNodeGroupDeleteFlags(fs *pflag.FlagSet) {
	fs.StringP("cluster", "k", c.ClusterName, "Name of the Kubernetes cluster")
}

func (c *NodeGroupNodeDeleteConfig) ValidateNodeGroupDeleteFlags(cmd *cobra.Command, args []string) error {
	flags.EnsureRequiredFlags(cmd, "cluster")
	return nil
}
