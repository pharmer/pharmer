package dokube

import (
	"time"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func (cm *ClusterManager) GetDefaultNodeSpec(cluster *api.Cluster, sku string) (api.NodeSpec, error) {
	return api.NodeSpec{
		SKU: sku,
	}, nil
}

func (cm *ClusterManager) SetDefaults(cluster *api.Cluster) error {
	n := namer{cluster: cluster}

	// Init object meta
	cluster.ObjectMeta.UID = uuid.NewUUID()
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()
	err := api.AssignTypeKind(cluster)
	if err != nil {
		return err
	}
	// Init spec
	cluster.Spec.Cloud.Region = cluster.Spec.Cloud.Zone
	cluster.Spec.Cloud.SSHKeyName = n.GenSSHKeyExternalID()

	cluster.Spec.Cloud.InstanceImage = "ubuntu-16-04-x64"
	cluster.Spec.Networking.SetDefaults()
	cluster.Spec.Networking.PodSubnet = "10.244.0.0/16"
	cluster.Spec.Networking.NetworkProvider = "CALICO"

	cluster.Spec.Cloud.Dokube = &api.DokubeSpec{}
	// Init status
	cluster.Status = api.ClusterStatus{
		Phase: api.ClusterPending,
	}
	return nil
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	return nil, errors.Errorf("Error ssh")
	/*cfg := &api.SSHConfig{
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
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil*/
}
