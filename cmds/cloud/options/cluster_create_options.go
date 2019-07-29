package options

import (
	"strings"
	"time"

	"github.com/appscode/go/flags"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/pkg/apis/core"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type ClusterCreateConfig struct {
	Cluster     *api.Cluster
	Nodes       map[string]int
	Namespace   string
	MasterCount int
}

func NewClusterCreateConfig() *ClusterCreateConfig {
	cluster := &api.Cluster{
		// Init object meta
		ObjectMeta: metav1.ObjectMeta{
			UID:               uuid.NewUUID(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Generation:        time.Now().UnixNano(),
		},
		Spec: api.PharmerClusterSpec{
			ClusterAPI: clusterapi.Cluster{},
			Config: api.ClusterConfig{
				Cloud: api.CloudSpec{
					NetworkProvider: api.PodNetworkCalico,
				},
			},
			AuditSink: false,
		},
	}

	return &ClusterCreateConfig{
		Namespace:   core.NamespaceDefault,
		Cluster:     cluster,
		Nodes:       map[string]int{},
		MasterCount: 1,
	}
}

func (c *ClusterCreateConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Cluster.Spec.Config.Cloud.CloudProvider, "provider", c.Cluster.Spec.Config.Cloud.CloudProvider, "Provider name")
	fs.StringVar(&c.Cluster.Spec.Config.Cloud.Zone, "zone", c.Cluster.Spec.Config.Cloud.Zone, "Cloud provider zone name")
	fs.StringVar(&c.Cluster.Spec.Config.CredentialName, "credential-uid", c.Cluster.Spec.Config.CredentialName, "Use preconfigured cloud credential uid")
	fs.StringVar(&c.Cluster.Spec.Config.KubernetesVersion, "kubernetes-version", c.Cluster.Spec.Config.KubernetesVersion, "Kubernetes version")
	fs.StringVar(&c.Cluster.Spec.Config.Cloud.NetworkProvider, "network-provider", c.Cluster.Spec.Config.Cloud.NetworkProvider, "Name of CNI plugin. Available options: calico, flannel, kubenet, weavenet")
	fs.BoolVar(&c.Cluster.Spec.AuditSink, "audit-sink", c.Cluster.Spec.AuditSink, "enable kubernetes audit sink")
	fs.StringVar(&c.Namespace, "namespace", c.Namespace, "Namespace")
	fs.IntVar(&c.MasterCount, "masters", c.MasterCount, "Number of masters")
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
	c.Cluster.Spec.ClusterAPI.Name = c.Cluster.Name
	c.Cluster.Spec.ClusterAPI.Namespace = c.Namespace
	c.Cluster.Spec.Config.MasterCount = c.MasterCount
	return nil
}
