package xorm

import (
	"fmt"
	"strings"
	"testing"
	"time"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func TestClusterEngine(t *testing.T) {
	engine, err := newPGEngine("postgres", "postgres", "127.0.0.1", 5432, "postgres")
	fmt.Println(err)
	x := New(engine)
	cluster := &api.Cluster{}
	cluster.Name = "xorm-test"
	cluster.Spec.Cloud.CloudProvider = "digitalocean"
	cluster.Spec.Cloud.Zone = "nyc3"
	cluster.Spec.CredentialName = "do"
	cluster.Spec.KubernetesVersion = "v1.8.0"

	cluster.Spec.Networking.NetworkProvider = "calico"
	// Init object meta
	cluster.ObjectMeta.UID = uuid.NewUUID()
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()
	api.AssignTypeKind(cluster)

	// Init spec
	cluster.Spec.Cloud.Region = cluster.Spec.Cloud.Zone
	cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	cluster.Spec.Cloud.InstanceImage = "ubuntu-16-04-x64"
	cluster.Spec.Networking.SetDefaults()
	cluster.Spec.AuthorizationModes = strings.Split(kubeadmapi.DefaultAuthorizationModes, ",")
	//cluster.Spec.APIServerCertSANs = ""
	cluster.Spec.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalIP),
			string(core.NodeExternalIP),
		}, ","),
	}
	// Init status
	cluster.Status = api.ClusterStatus{
		Phase: api.ClusterPending,
	}

	_, err = x.Clusters().Create(cluster)
	fmt.Println(err)
}
