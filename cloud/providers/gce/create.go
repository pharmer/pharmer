package gce

import (
	"net"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapiGCE "github.com/pharmer/pharmer/apis/v1beta1/gce"
	proconfig "github.com/pharmer/pharmer/apis/v1beta1/gce"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	cluster := cm.Cluster
	config := cluster.Spec.Config

	spec := clusterapiGCE.GCEMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: proconfig.GCEProviderGroupName + "/" + proconfig.GCEProviderApiVersion,
			Kind:       proconfig.GCEMachineProviderKind,
		},
		Zone:  config.Cloud.Zone,
		OS:    config.Cloud.InstanceImage,
		Roles: []api.MachineRole{role},
		Disks: []api.Disk{
			{
				InitializeParams: api.DiskInitializeParams{
					DiskType:   "pd-standard",
					DiskSizeGb: 30,
				},
			},
		},
		MachineType: sku,
	}

	rawSpec, err := clusterapiGCE.EncodeMachineSpec(&spec)
	if err != nil {
		return clusterapi.ProviderSpec{}, errors.Wrap(err, "Error encoding provider spec for gce cluster")
	}

	return clusterapi.ProviderSpec{
		Value:     rawSpec,
		ValueFrom: nil,
	}, nil
}

func (cm *ClusterManager) SetDefaultCluster() error {
	cluster := cm.Cluster

	n := namer{cluster: cluster}
	config := cluster.Spec.Config

	config.Cloud.InstanceImageProject = "ubuntu-os-cloud"
	config.Cloud.InstanceImage = "ubuntu-1604-xenial-v20170721"
	config.Cloud.OS = "ubuntu-1604-lts"
	config.Cloud.GCE = &api.GoogleSpec{
		NetworkName: "default",
		NodeTags:    []string{n.NodePrefix()},
	}

	config.APIServerExtraArgs["cloud-config"] = "/etc/kubernetes/ccm/cloud-config"

	config.KubeletExtraArgs = make(map[string]string)
	config.KubeletExtraArgs["cloud-provider"] = cluster.ClusterConfig().Cloud.CloudProvider // requires --cloud-config

	cluster.Spec.Config.Cloud.Region = cluster.Spec.Config.Cloud.Zone[0 : len(cluster.Spec.Config.Cloud.Zone)-2]
	config.ControllerManagerExtraArgs = map[string]string{
		"cloud-config":   "/etc/kubernetes/ccm/cloud-config",
		"cloud-provider": config.Cloud.CloudProvider,
	}

	if cluster.Spec.ClusterAPI.ObjectMeta.Annotations == nil {
		cluster.Spec.ClusterAPI.ObjectMeta.Annotations = make(map[string]string)
	}

	// set clusterAPI provider-specs
	return clusterapiGCE.SetGCEclusterProviderConfig(cluster.Spec.ClusterAPI, config.Cloud.Project, cm.Certs)
}

// TODO: why this is needed?
func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
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

// todo: is this needed?
func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, nil
}
