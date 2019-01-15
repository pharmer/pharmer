package options

import (
	"strings"
	"time"

	"github.com/appscode/go/flags"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type ClusterCreateConfig struct {
	Kube *api.Kube
	Cluster        *clusterapi.Cluster
	ProviderConfig *api.ClusterProviderConfig
	Nodes          map[string]int
	//Masters        int32
}

func NewClusterCreateConfig() *ClusterCreateConfig {
	kube := &api.Kube{
		ObjectMeta: metav1.ObjectMeta{
			UID: uuid.NewUUID(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Generation: time.Now().UnixNano(),
		},
		Spec: api.KubeSpec{},
	}
	cluster := &clusterapi.Cluster{
		// Init object meta
		ObjectMeta: metav1.ObjectMeta{
			UID:               uuid.NewUUID(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Generation:        time.Now().UnixNano(),
		},
		Spec: clusterapi.ClusterSpec{},
	}
	return &ClusterCreateConfig{
		Kube:kube,
		Cluster: cluster,
		ProviderConfig: &api.ClusterProviderConfig{
			NetworkProvider: "calico",
		},
		Nodes:   map[string]int{},
	//	Masters: 1,
	}
}

func (c *ClusterCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ProviderConfig.CloudProvider, "provider", c.ProviderConfig.CloudProvider, "Provider name")
	fs.StringVar(&c.ProviderConfig.Zone, "zone", c.ProviderConfig.Zone, "Cloud provider zone name")
	fs.StringVar(&c.Kube.Spec.CredentialName, "credential-uid", c.Kube.Spec.CredentialName, "Use preconfigured cloud credential uid")
	fs.StringVar(&c.Kube.Spec.KubernetesVersion, "kubernetes-version", c.Kube.Spec.KubernetesVersion, "Kubernetes version")
	fs.StringVar(&c.ProviderConfig.NetworkProvider, "network-provider", c.ProviderConfig.NetworkProvider, "Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet")

	fs.StringToIntVar(&c.Nodes, "nodes", c.Nodes, "Node set configuration")
	//fs.Int32Var(&c.Masters, "masters", c.Masters, "Node set configuration")

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
	c.Kube.Name =c.Cluster.Name
	return nil
}
