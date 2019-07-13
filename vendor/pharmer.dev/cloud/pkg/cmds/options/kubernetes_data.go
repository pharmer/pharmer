package options

import (
	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	AllProvider string = "all_provider"
)

type KubernetesData struct {
	Provider   string
	Version    string
	Envs       []string
	Deprecated bool
}

func NewKubernetesData() *KubernetesData {
	return &KubernetesData{
		Provider: AllProvider,
	}
}

func (c *KubernetesData) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider (If this flag is not provided, then changes will apply to all supported cloud providers)")
	fs.StringVar(&c.Version, "version", c.Version, "kubernetes version (required)")
	fs.StringSliceVar(&c.Envs, "env", c.Envs, "environment variable (required, Example: --env=dev,qa,prod)")
	fs.BoolVar(&c.Deprecated, "deprecated", c.Deprecated, "To indicate whether provided environment variables are deprecated or not")
}

func (c *KubernetesData) ValidateFlags(cmd *cobra.Command, args []string) error {
	var ensureFlags = []string{
		"version",
		"env",
	}

	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	return nil
}
