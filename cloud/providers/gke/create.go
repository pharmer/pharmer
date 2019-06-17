package gke

import (
	"encoding/json"
	"net"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apis/v1beta1/gce"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
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
	n := namer{cluster: cluster}
	config := &cluster.Spec.Config

	config.Cloud.InstanceImage = "Ubuntu"
	config.Cloud.GKE = &api.GKESpec{
		UserName:    n.AdminUsername(),
		Password:    rand.GeneratePassword(),
		NetworkName: "default",
	}

	return nil
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, cloud.ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	n := namer{cluster: cluster}
	cfg := &api.SSHConfig{
		PrivateKey: cm.Certs.SSHKey.PrivateKey,
		User:       n.AdminUsername(),
		HostPort:   int32(22),
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == core.NodeExternalIP {
			cfg.HostIP = addr.Address
		}
	}
	if net.ParseIP(cfg.HostIP) == nil {
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}
