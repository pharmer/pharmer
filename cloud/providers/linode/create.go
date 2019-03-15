package linode

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	linodeconfig "github.com/pharmer/pharmer/apis/v1beta1/linode"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	roles := []api.MachineRole{api.NodeRole}
	if sku == "" {
		sku = "g6-standard-2"
		roles = []api.MachineRole{api.MasterRole}
	}
	config := cluster.Spec.Config
	spec := &linodeconfig.LinodeMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: linodeconfig.LinodeProviderGroupName + "/" + linodeconfig.LinodeProviderApiVersion,
			Kind:       linodeconfig.LinodeProviderKind,
		},
		Roles:  roles,
		Region: config.Cloud.Region,
		Type:   sku,
		Image:  config.Cloud.InstanceImage,
		Pubkey: string(SSHKey(cm.ctx).PublicKey),
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
	cluster.Spec.Config.Cloud.Region = cluster.Spec.Config.Cloud.Zone
	cluster.Spec.Config.Cloud.SSHKeyName = n.GenSSHKeyExternalID()
	config.Cloud.InstanceImage = "linode/ubuntu16.04lts"
	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)
	config.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	config.APIServerExtraArgs = map[string]string{
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalIP),
			string(core.NodeExternalIP),
		}, ","),
	}

	cluster.ClusterConfig().Cloud.CCMCredentialName = cluster.ClusterConfig().CredentialName
	cluster.ClusterConfig().Cloud.Linode = &api.LinodeSpec{
		RootPassword: rand.GeneratePassword(),
	}

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}
	cluster.SetNetworkingDefaults("calico")
	return linodeconfig.SetLinodeClusterProviderConfig(cluster.Spec.ClusterAPI, config)
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
