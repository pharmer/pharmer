package options

import (
	"strings"

	"github.com/appscode/go/flags"
	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeSSHConfig struct {
	ClusterName string
	NodeName    string
	Owner       string
}

func NewNodeSSHConfig() *NodeSSHConfig {
	return &NodeSSHConfig{
		ClusterName: "",
		NodeName:    "",
		Owner:       utils.GetLocalOwner(),
	}
}

func (c *NodeSSHConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of cluster")
	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")
}

func (c *NodeSSHConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	flags.EnsureRequiredFlags(cmd, "cluster")

	if len(args) == 0 {
		return errors.New("missing node name")
	}
	if len(args) > 1 {
		return errors.New("multiple node name provided")
	}
	c.ClusterName = strings.ToLower(c.ClusterName)
	c.NodeName = args[0]
	return nil
}
