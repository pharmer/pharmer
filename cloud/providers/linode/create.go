package linode

import (
	"encoding/json"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	linodeconfig "github.com/pharmer/pharmer/apis/v1beta1/linode"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (v1alpha1.ProviderSpec, error) {
	log := cm.Logger
	cluster := cm.Cluster

	roles := []api.MachineRole{api.NodeMachineRole}
	if sku == "" {
		sku = "g6-standard-2"
		roles = []api.MachineRole{api.MasterMachineRole}
	}
	config := cluster.Spec.Config

	pubkey, _, err := cm.StoreProvider.SSHKeys(cluster.Name).Get(cluster.GenSSHKeyExternalID())
	if err != nil {
		log.Error(err, "failed to get ssh keys from store")
		return clusterapi.ProviderSpec{}, err
	}

	spec := &linodeconfig.LinodeMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: linodeconfig.LinodeProviderGroupName + "/" + linodeconfig.LinodeProviderAPIVersion,
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
		log.Error(err, "failed to marshal provider spec to json")
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

	config.Cloud.InstanceImage = "linode/ubuntu16.04lts"
	config.SSHUserName = "root"

	config.Cloud.Linode = &api.LinodeSpec{
		RootPassword: rand.GeneratePassword(),
	}

	return linodeconfig.SetLinodeClusterProviderConfig(&cluster.Spec.ClusterAPI)
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	return kube.GetAdminConfig(cm.Cluster, cm.GetCaCertPair())
}
