package gke

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	spec := &api.GKEMachineProviderSpec{
		Zone:        cluster.Spec.Config.Cloud.Zone,
		MachineType: sku,
		Roles:       []api.MachineRole{role},
		Disks: []api.Disk{
			{
				InitializeParams: api.DiskInitializeParams{
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

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) SetDefaultCluster(cluster *api.Cluster, config *api.ClusterConfig) error {
	n := namer{cluster: cluster}

	if err := api.AssignTypeKind(cluster); err != nil {
		return err
	}
	if err := api.AssignTypeKind(cluster.Spec.ClusterAPI); err != nil {
		return err
	}

	// Init spec
	config.Cloud.Region = config.Cloud.Zone[0:strings.LastIndex(config.Cloud.Zone, "-")]
	config.Cloud.SSHKeyName = n.GenSSHKeyExternalID()
	//cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight
	// PREEMPTIBLE_NODE = false // Removed Support

	//cluster.Spec.Cloud.InstanceImageProject = "ubuntu-os-cloud"
	config.Cloud.InstanceImage = "Ubuntu"
	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)

	config.Cloud.GKE = &api.GKESpec{
		UserName:    n.AdminUsername(),
		Password:    rand.GeneratePassword(),
		NetworkName: "default",
	}
	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}
	return cluster.SetGKEProviderConfig(cluster.Spec.ClusterAPI, config)
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	n := namer{cluster: cluster}
	cfg := &api.SSHConfig{
		PrivateKey: SSHKey(cm.ctx).PrivateKey,
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
