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
package gke

import (
	"encoding/json"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/apis/v1alpha1/gce"
	"pharmer.dev/pharmer/cloud"

	"github.com/appscode/go/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (v1alpha1.ProviderSpec, error) {
	log := cm.Logger

	cluster := cm.Cluster
	spec := &gce.GCEMachineProviderSpec{
		Zone:        cluster.Spec.Config.Cloud.Zone,
		MachineType: sku,
		Roles:       []api.MachineRole{role},
		Disks: []gce.Disk{
			{
				InitializeParams: gce.DiskInitializeParams{
					DiskSizeGb: 100,
					DiskType:   "pd-standard",
				},
			},
		},
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

	config.Cloud.InstanceImage = "Ubuntu"
	config.SSHUserName = cm.namer.AdminUsername()

	config.Cloud.GKE = &api.GKESpec{
		UserName:    cm.namer.AdminUsername(),
		Password:    rand.GeneratePassword(),
		NetworkName: "default",
	}

	return nil
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, cloud.ErrNotImplemented
}
