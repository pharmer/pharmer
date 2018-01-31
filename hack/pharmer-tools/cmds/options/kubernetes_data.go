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
	Provider string
	Version  string
	Envs     string
}

func NewKubernetesData() *KubernetesData {
	return &KubernetesData{
		Provider: AllProvider,
	}
}

func (c *KubernetesData) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Provider, "provider", "p", c.Provider, "Name of the Cloud provider (If this flag is not provided, then changes will apply to all supported cloud providers)")
	fs.StringVar(&c.Version, "version", c.Version, "kubernetes version (required)")
	fs.StringVar(&c.Envs, "env", c.Envs, "environment variable, if this flag is empty or not provided then kubernetes version support will be deleted (Example: --env=dev,qa,prod)")
}

func (c *KubernetesData) ValidateFlags(cmd *cobra.Command, args []string) error {
	var ensureFlags = []string{
		"version",
	}

	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	return nil
}
