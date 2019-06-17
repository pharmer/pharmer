package dokube

import (
	"encoding/json"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	dokube_config "github.com/pharmer/pharmer/apis/v1beta1/dokube"
	"github.com/pharmer/pharmer/cloud"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	spec := &dokube_config.DokubeMachineProviderConfig{
		Size: sku,
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

	cluster.Spec.Config.Cloud.InstanceImage = "ubuntu-16-04-x64"
	cluster.Spec.Config.Cloud.Dokube = &api.DokubeSpec{}

	return dokube_config.SetLDokubeClusterProviderConfig(&cluster.Spec.ClusterAPI, nil)
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, cloud.ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	return nil, cloud.ErrNotImplemented
}
