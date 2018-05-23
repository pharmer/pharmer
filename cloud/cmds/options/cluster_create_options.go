package options

import (
	"time"

	"github.com/appscode/go/flags"
	api "github.com/pharmer/pharmer/apis/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

type ClusterCreateConfig struct {
	Cluster        *api.Cluster
	ProviderConfig *api.ClusterProviderConfig
	Nodes          map[string]int
	HaNode         int32
}

func NewClusterCreateConfig() *ClusterCreateConfig {
	cluster := &api.Cluster{
		// Init object meta
		ObjectMeta: metav1.ObjectMeta{
			UID:               uuid.NewUUID(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Generation:        time.Now().UnixNano(),
		},
		Spec: api.PharmerClusterSpec{},
	}
	return &ClusterCreateConfig{
		Cluster: cluster,
		ProviderConfig: &api.ClusterProviderConfig{
			NetworkProvider: "calico",
		},
		Nodes:  map[string]int{},
		HaNode: 1,
	}
}

func (c *ClusterCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ProviderConfig.CloudProvider, "provider", c.ProviderConfig.CloudProvider, "Provider name")
	fs.StringVar(&c.ProviderConfig.Zone, "zone", c.ProviderConfig.Zone, "Cloud provider zone name")
	fs.StringVar(&c.Cluster.Spec.CredentialName, "credential-uid", c.Cluster.Spec.CredentialName, "Use preconfigured cloud credential uid")
	fs.StringVar(&c.Cluster.Spec.KubernetesVersion, "kubernetes-version", c.Cluster.Spec.KubernetesVersion, "Kubernetes version")
	fs.StringVar(&c.ProviderConfig.NetworkProvider, "network-provider", c.ProviderConfig.NetworkProvider, "Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet")

	fs.StringToIntVar(&c.Nodes, "nodes", c.Nodes, "Node set configuration")
	fs.Int32Var(&c.HaNode, "ha-node", c.HaNode, "Node set configuration")

}

func (c *ClusterCreateConfig) ValidateFlags(cmd *cobra.Command, args []string) error {
	ensureFlags := []string{"provider", "zone", "kubernetes-version"}
	flags.EnsureRequiredFlags(cmd, ensureFlags...)

	if len(args) == 0 {
		errors.New("missing cluster name")
	}
	if len(args) > 1 {
		errors.New("multiple cluster name provided")
	}
	c.Cluster.Name = args[0]
	return nil
}
