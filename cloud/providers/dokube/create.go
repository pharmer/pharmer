/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package dokube

import (
	"encoding/json"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	dokube_config "pharmer.dev/pharmer/apis/v1alpha1/dokube"
	"pharmer.dev/pharmer/cloud"

	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	spec := &dokube_config.DokubeMachineProviderConfig{
		Size: sku,
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

	cluster.Spec.Config.Cloud.InstanceImage = "ubuntu-16-04-x64"
	cluster.Spec.Config.Cloud.Dokube = &api.DokubeSpec{}
	cluster.Spec.Config.SSHUserName = "root"

	return dokube_config.SetLDokubeClusterProviderConfig(&cluster.Spec.ClusterAPI, nil)
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, cloud.ErrNotImplemented
}
