package softlayer

import (
	"fmt"
	"net"
	"strings"
	"time"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func (cm *ClusterManager) GetDefaultNodeSpec(sku string) (api.NodeSpec, error) {
	if sku == "" {
		sku = "2c2m"
	}
	return api.NodeSpec{
		SKU: sku,
		//	DiskType:      "gp2",
		//	DiskSize:      100,
	}, nil
}
func (cm *ClusterManager) SetDefaults(cluster *api.Cluster) error {
	n := namer{cluster: cluster}

	// Init object meta
	cluster.ObjectMeta.UID = uuid.NewUUID()
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()
	api.AssignTypeKind(cluster)

	// Init spec
	cluster.Spec.Cloud.CCMCredentialName = cluster.Spec.CredentialName
	cluster.Spec.Cloud.Region = cluster.Spec.Cloud.Zone[:len(cluster.Spec.Cloud.Zone)-2]
	cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	cluster.Spec.Networking.SetDefaults()
	cluster.Spec.Cloud.InstanceImage = "UBUNTU_16_64"
	cluster.Spec.AuthorizationModes = strings.Split(kubeadmapi.DefaultAuthorizationModes, ",")
	cluster.Spec.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	cluster.Spec.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalIP),
			string(core.NodeExternalIP),
		}, ","),
	}
	// kubelet log: error: failed to run Kubelet: Running with swap on is not supported, please disable swap! or set --fail-swap-on flag to false.
	// https://github.com/kubernetes/kubernetes/issues/50373
	// https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-335219173
	cluster.Spec.KubeletExtraArgs = map[string]string{
		"fail-swap-on": "false",
	}

	// Init status
	cluster.Status = api.ClusterStatus{
		Phase:            api.ClusterPending,
		SSHKeyExternalID: n.GenSSHKeyExternalID(),
	}
	cluster.Spec.Networking.NonMasqueradeCIDR = "10.0.0.0/8"

	return nil
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	cfg := &api.SSHConfig{
		PrivateKey: SSHKey(cm.ctx).PrivateKey,
		User:       "root",
		HostPort:   int32(22),
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == core.NodeExternalIP {
			cfg.HostIP = addr.Address
		}
	}
	if net.ParseIP(cfg.HostIP) == nil {
		return nil, fmt.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}
