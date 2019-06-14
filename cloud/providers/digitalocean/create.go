package digitalocean

import (
	"encoding/json"
	"net"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	doCapi "github.com/pharmer/pharmer/apis/v1beta1/digitalocean"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	cluster := cm.Cluster
	if sku == "" {
		sku = "2gb"
	}
	config := cluster.Spec.Config

	pubkey, _, err := store.StoreProvider.SSHKeys(cluster.Name).Get(cluster.GenSSHKeyExternalID())
	if err != nil {
		return clusterapi.ProviderSpec{}, errors.Wrap(err, " failed to get ssh keys")
	}

	spec := &doCapi.DigitalOceanMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: doCapi.DigitalOceanProviderGroupName + "/" + doCapi.DigitalOceanProviderApiVersion,
			Kind:       doCapi.DigitalOceanProviderKind,
		},
		Region: config.Cloud.Region,
		Size:   sku,
		Image:  config.Cloud.InstanceImage,
		Tags:   []string{"KubernetesCluster:" + cluster.Name},
		SSHPublicKeys: []string{
			string(pubkey),
		},
		PrivateNetworking: true,
		Backups:           false,
		IPv6:              false,
		Monitoring:        true,
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

	config.Cloud.InstanceImage = "ubuntu-18-04-x64"

	return doCapi.SetDigitalOceanClusterProviderConfig(&cluster.Spec.ClusterAPI)
}

// IsValid TODO: Add Description
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
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, nil
}
