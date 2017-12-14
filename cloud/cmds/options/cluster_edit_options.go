package options

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterEditConfig struct {
	ClusterName       string
	File              string
	KubernetesVersion string
	KubeletVersion    string
	KubeadmVersion    string
	Locked            bool
	Output            string
}

func NewClusterEditConfig() *ClusterEditConfig {
	return &ClusterEditConfig{
		ClusterName:       "",
		File:              "",
		KubernetesVersion: "",
		KubeletVersion:    "",
		KubeadmVersion:    "",
		Locked:            false,
		Output:            "yaml",
	}
}

func (c *ClusterEditConfig) AddClusterEditFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.File, "file", "f", c.File, "Load cluster data from file")
	//TODO: Add necessary flags that will be used for update
	fs.StringVar(&c.KubernetesVersion, "kubernetes-version", c.KubernetesVersion, "Kubernetes version")
	fs.StringVar(&c.KubeletVersion, "kubelet-version", c.KubeletVersion, "kubelet/kubectl version")
	fs.StringVar(&c.KubeadmVersion, "kubeadm-version", c.KubeadmVersion, "Kubeadm version")
	fs.BoolVar(&c.Locked, "locked", c.Locked, "If true, locks cluster from deletion")
	fs.StringVarP(&c.Output, "output", "o", c.Output, "Output format. One of: yaml|json.")

}

func (c *ClusterEditConfig) ValidateClusterEditFlags(cmd *cobra.Command, args []string) error {
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
	c.ClusterName = args[0]
	return nil
}

func (c *ClusterEditConfig) CheckForUpdateFlags() bool {
	if c.Locked || c.KubernetesVersion != "" ||
		c.KubeletVersion != "" || c.KubeadmVersion != "" {
		return true
	}
	return false
}
