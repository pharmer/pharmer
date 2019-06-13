package linode

import (
	"encoding/json"
	"net"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	linodeconfig "github.com/pharmer/pharmer/apis/v1beta1/linode"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (v1alpha1.ProviderSpec, error) {
	cluster := cm.Cluster

	roles := []api.MachineRole{api.NodeMachineRole}
	if sku == "" {
		sku = "g6-standard-2"
		roles = []api.MachineRole{api.MasterMachineRole}
	}
	config := cluster.Spec.Config

	pubkey, _, err := store.StoreProvider.SSHKeys(cluster.Name).Get(cluster.GenSSHKeyExternalID())
	if err != nil {
		return clusterapi.ProviderSpec{}, errors.Wrap(err, " failed to get ssh keys")
	}

	spec := &linodeconfig.LinodeMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: linodeconfig.LinodeProviderGroupName + "/" + linodeconfig.LinodeProviderApiVersion,
			Kind:       linodeconfig.LinodeProviderKind,
		},
		Roles:  roles,
		Region: config.Cloud.Region,
		Type:   sku,
		Image:  config.Cloud.InstanceImage,
		Pubkey: string(pubkey),
	}

	providerSpecValue, err := json.Marshal(spec)
	if err != nil {
		return clusterapi.ProviderSpec{}, err
	}

	return clusterapi.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: providerSpecValue,
		},
	}, nil
}

func (cm *ClusterManager) SetDefaultCluster() error {
	cluster := cm.Cluster
	config := cluster.Spec.Config

	config.Cloud.InstanceImage = "linode/ubuntu16.04lts"
	config.Cloud.Linode = &api.LinodeSpec{
		RootPassword: rand.GeneratePassword(),
	}

	return linodeconfig.SetLinodeClusterProviderConfig(&cluster.Spec.ClusterAPI)
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	cfg := &api.SSHConfig{
		PrivateKey: cm.Certs.SSHKey.PrivateKey,
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
	return cfg, nil
}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, nil
}
