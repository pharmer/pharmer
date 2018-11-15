package options

import (
	"strings"

	"github.com/appscode/go/flags"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ClusterCreateConfig struct {
	Cluster *api.Cluster
	Nodes   map[string]int
	Owner   string
}

func NewClusterCreateConfig() *ClusterCreateConfig {
	return &ClusterCreateConfig{
		Cluster: &api.Cluster{
			Spec: api.ClusterSpec{
				Networking: api.Networking{
					NetworkProvider: "calico",
				},
			},
		},
		Nodes: map[string]int{},
		Owner: utils.GetLocalOwner(),
	}
}

func (c *ClusterCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Cluster.Spec.Cloud.CloudProvider, "provider", c.Cluster.Spec.Cloud.CloudProvider, "Provider name")
	fs.StringVar(&c.Cluster.Spec.Cloud.Zone, "zone", c.Cluster.Spec.Cloud.Zone, "Cloud provider zone name")
	fs.StringVar(&c.Cluster.Spec.CredentialName, "credential-uid", c.Cluster.Spec.CredentialName, "Use preconfigured cloud credential uid")
	fs.StringVar(&c.Cluster.Spec.KubernetesVersion, "kubernetes-version", c.Cluster.Spec.KubernetesVersion, "Kubernetes version")
	fs.StringVar(&c.Cluster.Spec.Networking.NetworkProvider, "network-provider", c.Cluster.Spec.Networking.NetworkProvider, "Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet")

	fs.StringToIntVar(&c.Nodes, "nodes", c.Nodes, "Node set configuration")
	fs.StringVarP(&c.Owner, "owner", "o", c.Owner, "Current user id")

}

func (c *ClusterCreateConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	ensureFlags := []string{"provider", "zone", "kubernetes-version"}
	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	if len(args) == 0 {
		return errors.New("missing cluster name")
	}
	if len(args) > 1 {
		return errors.New("multiple cluster name provided")
	}
	c.Cluster.Name = strings.ToLower(args[0])
	return nil
}
