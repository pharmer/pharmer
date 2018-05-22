package options

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeGroupGetConfig struct {
	ClusterName string
	Output      string
	NodeGroups  []string
}

func NewNodeGroupGetConfig() *NodeGroupGetConfig {
	return &NodeGroupGetConfig{
		ClusterName: "",
		Output:      "",
	}
}

func (c *NodeGroupGetConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of the Kubernetes cluster")
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: json|yaml|wide")

}

func (c *NodeGroupGetConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	c.ClusterName = strings.ToLower(c.ClusterName)
	c.NodeGroups = args
	return nil
}
