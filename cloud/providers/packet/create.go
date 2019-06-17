package packet

import (
	"encoding/json"
	"net"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	packetconfig "github.com/pharmer/pharmer/apis/v1beta1/packet"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	if sku == "" {
		/*mts, err := cm.conn.i.ListMachineTypes()
		if err != nil {
			return clusterapi.ProviderSpec{}, err
		}

		var ins *cloudapi.MachineType
		for _, instance := range mts {
			zones := sets.NewString(instance.Spec.Zones...)
			if zones.Has(cluster.ClusterConfig().Cloud.Zone) && instance.Spec.CPU.CmpInt64(2) >= 0 {
				ins = &instance
				break
			}
		}
		if ins == nil {
			return clusterapi.ProviderSpec{}, errors.Errorf("can't find instance for provider %v with zone %v and cpu %v", cluster.Spec.Config.Cloud.CloudProvider, cluster.ClusterConfig().Cloud.Zone, 2)
		}
		sku = ins.Spec.SKU*/

		// TODO: lets fix this in next release
		sku = "baremetal_0"
	}
	//config := cluster.Spec.Config
	spec := &packetconfig.PacketMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: packetconfig.PacketProviderGroupName + "/" + packetconfig.PacketProviderAPIVersion,
			Kind:       packetconfig.PacketProviderKind,
		},
		Plan:         sku,
		SpotInstance: "Regular",
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
	config := &cluster.Spec.Config

	config.Cloud.InstanceImage = "ubuntu_16_04" // 1b9b78e3-de68-466e-ba00-f2123e89c112

	return packetconfig.SetPacketClusterProviderConfig(&cluster.Spec.ClusterAPI)
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, cloud.ErrNotImplemented
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

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	return nil, nil
}
