package options

import (
	"strings"

	"github.com/appscode/go/flags"
	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type NodeGroupEditConfig struct {
	ClusterName string
	NgName      string
	File        string
	DoNotDelete bool
	Output      string
	Owner       string
}

func NewNodeGroupEditConfig() *NodeGroupEditConfig {
	return &NodeGroupEditConfig{
		ClusterName: "",
		NgName:      "",
		File:        "",
		DoNotDelete: false,
		Output:      "yaml",
		Owner:       utils.GetLocalOwner(),
	}
}

func (c *NodeGroupEditConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.ClusterName, "cluster", "k", c.ClusterName, "Name of the Kubernetes cluster")
	fs.StringVarP(&c.File, "file", "f", c.File, "Load nodegroup data from file")
	fs.BoolVarP(&c.DoNotDelete, "do-not-delete", "", c.DoNotDelete, "Set do not delete flag")
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: yaml|json.")
	fs.StringVarP(&c.Owner, "owner", "", c.Owner, "Current user id")
}

func (c *NodeGroupEditConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	flags.EnsureRequiredFlags(cmd, "cluster")
	if len(args) == 0 {
		return errors.New("missing nodegroup name")
	}
	if len(args) > 1 {
		return errors.New("multiple nodegroup name provided")
	}
	c.ClusterName = strings.ToLower(c.ClusterName)
	c.NgName = args[0]
	return nil
}
