package options

import (
	"errors"

	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeSSHConfig struct {
	ClusterName string
	NodeName    string
}

func NewNodeSSHConfig() *NodeSSHConfig {
	return &NodeSSHConfig{
		ClusterName: "",
		NodeName:    "",
	}
}

func (c *NodeSSHConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of cluster")
}

func (c *NodeSSHConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	flags.EnsureRequiredFlags(cmd, "cluster")

	if len(args) == 0 {
		errors.New("missing node name")
	}
	if len(args) > 1 {
		errors.New("multiple node name provided")
	}
	c.NodeName = args[0]

	return nil
}
