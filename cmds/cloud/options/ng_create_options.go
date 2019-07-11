package options

import (
	"strings"

	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeGroupCreateConfig struct {
	ClusterName string
	Nodes       map[string]int
}

func NewNodeGroupCreateConfig() *NodeGroupCreateConfig {
	return &NodeGroupCreateConfig{
		ClusterName: "",
		Nodes:       map[string]int{},
	}
}

func (c *NodeGroupCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of the Kubernetes cluster")
	fs.StringToIntVar(&c.Nodes, "nodes", c.Nodes, "Node set configuration")
}

func (c *NodeGroupCreateConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	ensureFlags := []string{"cluster", "nodes"}
	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	c.ClusterName = strings.ToLower(c.ClusterName)
	return nil
}