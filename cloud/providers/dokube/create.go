package dokube

import (
	"encoding/json"
	"time"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apis/v1beta1/dokube"
	. "github.com/pharmer/pharmer/cloud"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
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

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) SetDefaultCluster(cluster *api.Cluster, config *api.ClusterConfig) error {
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
	cluster.Spec.Config.Cloud.Region = cluster.Spec.Config.Cloud.Zone
	cluster.Spec.Config.Cloud.SSHKeyName = n.GenSSHKeyExternalID()

	cluster.Spec.Config.Cloud.InstanceImage = "ubuntu-16-04-x64"
	cluster.SetNetworkingDefaults(cluster.Spec.Config.Cloud.NetworkProvider)

	cluster.Spec.Config.Cloud.Dokube = &api.DokubeSpec{}
	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}

	cluster.SetNetworkingDefaults("calico")

	return dokube_config.SetLDokubeClusterProviderConfig(cluster.Spec.ClusterAPI, nil)
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	return nil, ErrNotImplemented
}
