package options

import (
	"strings"

	"github.com/pharmer/pharmer/utils"

	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeGroupCreateConfig struct {
	ClusterName string
	Nodes       map[string]int
	Owner       string
}

func NewNodeGroupCreateConfig() *NodeGroupCreateConfig {
	return &NodeGroupCreateConfig{
		ClusterName: "",
		Nodes:       map[string]int{},
		Owner:       utils.GetLocalOwner(),
	}
}

func (c *NodeGroupCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of the Kubernetes cluster")
	fs.StringToIntVar(&c.Nodes, "nodes", c.Nodes, "Node set configuration")

	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")
}

func (c *NodeGroupCreateConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	ensureFlags := []string{"cluster", "nodes"}
	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	c.ClusterName = strings.ToLower(c.ClusterName)
	return nil
}
