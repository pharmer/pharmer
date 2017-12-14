package options

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterGetConfig struct {
	Clusters []string
	Output   string
}

func NewClusterGetConfig() *ClusterGetConfig {
	return &ClusterGetConfig{
		Output: "",
	}
}

func (c *ClusterGetConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: json|yaml|wide")
}

func (c *ClusterGetConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	c.Clusters = args
	return nil
}
