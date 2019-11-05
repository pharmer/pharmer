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
package digitalocean

import (
	"encoding/json"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	doCapi "pharmer.dev/pharmer/apis/v1alpha1/digitalocean"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cloud/utils/kube"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	log := cm.Logger
	cluster := cm.Cluster
	if sku == "" {
		sku = "2gb"
	}
	config := cluster.Spec.Config

	pubkey, _, err := cm.StoreProvider.SSHKeys(cluster.Name).Get(cluster.GenSSHKeyExternalID())
	if err != nil {
		log.Error(err, "failed to get ssh keys")
		return clusterapi.ProviderSpec{}, err
	}

	spec := &doCapi.DigitalOceanMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: doCapi.DigitalOceanProviderGroupName + "/" + doCapi.DigitalOceanProviderAPIVersion,
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

	config.Cloud.InstanceImage = "ubuntu-18-04-x64"
	config.Cloud.Region = config.Cloud.Zone
	config.SSHUserName = "root"

	return doCapi.SetDigitalOceanClusterProviderConfig(&cluster.Spec.ClusterAPI)
}

// IsValid TODO: Add Description
func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, cloud.ErrNotImplemented
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	return kube.GetAdminConfig(cm.Cluster, cm.GetCaCertPair())
}
