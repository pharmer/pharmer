package gke

import (
	"encoding/json"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apis/v1beta1/gce"
	"github.com/pharmer/pharmer/cloud"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (v1alpha1.ProviderSpec, error) {
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
