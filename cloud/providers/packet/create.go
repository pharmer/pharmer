package packet

import (
	"encoding/json"
	"net"
	"strings"

	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	packetconfig "github.com/pharmer/pharmer/apis/v1beta1/packet"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	if sku == "" {
		mts, err := cm.conn.i.ListMachineTypes()
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
		sku = ins.Spec.SKU
	}
	//config := cluster.Spec.Config
	spec := &packetconfig.PacketMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: packetconfig.PacketProviderGroupName + "/" + packetconfig.PacketProviderApiVersion,
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

func (cm *ClusterManager) SetDefaultCluster(cluster *api.Cluster, config *api.ClusterConfig) error {
	n := namer{cluster: cluster}

	if err := api.AssignTypeKind(cluster); err != nil {
		return err
	}
	if err := api.AssignTypeKind(cluster.Spec.ClusterAPI); err != nil {
		return err
	}
	// Init spec
	cluster.ClusterConfig().Cloud.Region = cluster.ClusterConfig().Cloud.Zone
	cluster.ClusterConfig().Cloud.SSHKeyName = n.GenSSHKeyExternalID()
	//cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	cluster.ClusterConfig().Cloud.CCMCredentialName = cluster.ClusterConfig().CredentialName
	cluster.ClusterConfig().Cloud.InstanceImage = "ubuntu_16_04" // 1b9b78e3-de68-466e-ba00-f2123e89c112
	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)
	cluster.ClusterConfig().APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	cluster.ClusterConfig().APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalIP),
			string(core.NodeExternalIP),
		}, ","),
	}

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}
	cluster.SetNetworkingDefaults("calico")

	return packetconfig.SetPacketClusterProviderConfig(cluster.Spec.ClusterAPI)
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
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, nil
}
