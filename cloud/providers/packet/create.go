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
package packet

import (
	"encoding/json"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	packetconfig "pharmer.dev/pharmer/apis/v1alpha1/packet"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cloud/utils/kube"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	log := cm.Logger
	if sku == "" {
		sku = "baremetal_0"
	}
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
		log.Error(err, "failed to marshal provider spec")
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
	config.SSHUserName = "root"

	return packetconfig.SetPacketClusterProviderConfig(&cluster.Spec.ClusterAPI)
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, cloud.ErrNotImplemented
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	return kube.GetAdminConfig(cm.Cluster, cm.GetCaCertPair())
}
