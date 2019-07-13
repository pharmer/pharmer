package eks

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/apis/v1beta1/aws"
	"pharmer.dev/pharmer/cloud"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	spec := &aws.AWSMachineProviderSpec{
		InstanceType: sku,
	}

	providerSpecValue, err := json.Marshal(spec)
	if err != nil {
		cm.Logger.Error(err, "failed to marshal provider spec")
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

	config.SSHUserName = "ubuntu"
	cluster.Status.Cloud = api.CloudStatus{
		EKS: &api.EKSStatus{},
	}

	return nil
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, cloud.ErrNotImplemented
}
