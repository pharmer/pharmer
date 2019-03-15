package options

import (
	"strings"

	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterEditConfig struct {
	ClusterName       string
	File              string
	KubernetesVersion string
	Locked            bool
	Output            string
	Owner             string
}

func NewClusterEditConfig() *ClusterEditConfig {
	return &ClusterEditConfig{
		ClusterName:       "",
		File:              "",
		KubernetesVersion: "",
		Locked:            false,
		Output:            "yaml",
		Owner:             utils.GetLocalOwner(),
	}
}

func (c *ClusterEditConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.File, "file", "f", c.File, "Load cluster data from file")
	//TODO: Add necessary flags that will be used for update
	fs.StringVar(&c.KubernetesVersion, "kubernetes-version", c.KubernetesVersion, "Kubernetes version")
	fs.BoolVar(&c.Locked, "locked", c.Locked, "If true, locks cluster from deletion")
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: yaml|json.")
	fs.StringVarP(&c.Owner, "owner", "", c.Owner, "Current user id")

}

func (c *ClusterEditConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("file") {
		if len(args) != 0 {
			return errors.New("no argument can be provided when --file flag is used")
		}
	}
	if len(args) == 0 {
		return errors.New("missing cluster name")
	}
	if len(args) > 1 {
		return errors.New("multiple cluster name provided")
	}
	c.ClusterName = strings.ToLower(args[0])
	return nil
}

func (c *ClusterEditConfig) CheckForUpdateFlags() bool {
	if c.Locked || c.KubernetesVersion != "" {
		return true
	}
	return false
}
