package gke

import (
	"encoding/json"

	"github.com/appscode/go/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/apis/v1beta1/gce"
	"pharmer.dev/pharmer/cloud"
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
