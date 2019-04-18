package digitalocean

import (
	"encoding/json"
	"net"
	"strings"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	doCapi "github.com/pharmer/pharmer/apis/v1beta1/digitalocean"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	if sku == "" {
		sku = "2gb"
	}
	config := cluster.Spec.Config
	spec := &doCapi.DigitalOceanMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: doCapi.DigitalOceanProviderGroupName + "/" + doCapi.DigitalOceanProviderApiVersion,
			Kind:       doCapi.DigitalOceanProviderKind,
		},
		Region: config.Cloud.Region,
		Size:   sku,
		Image:  config.Cloud.InstanceImage,
		Tags:   []string{"KubernetesCluster:" + cluster.Name},
		SSHPublicKeys: []string{
			string(SSHKey(cm.ctx).PublicKey),
		},
		PrivateNetworking: true,
		Backups:           false,
		IPv6:              false,
		Monitoring:        true,
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

func (cm *ClusterManager) SetDefaultCluster(cluster *api.Cluster, config *api.ClusterConfig) error {
	n := namer{cluster: cluster}

	if err := api.AssignTypeKind(cluster); err != nil {
		return err
	}
	if err := api.AssignTypeKind(cluster.Spec.ClusterAPI); err != nil {
		return err
	}
	config.Cloud.Region = config.Cloud.Zone
	config.Cloud.SSHKeyName = n.GenSSHKeyExternalID()
	config.Cloud.InstanceImage = "ubuntu-18-04-x64"

	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)
	config.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	config.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeExternalDNS),
			string(core.NodeExternalIP),
			string(core.NodeInternalIP),
		}, ","),
		//	"endpoint-reconciler-type": "lease",
	}

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}
	cm.cluster = cluster
	cluster.SetNetworkingDefaults("calico")
	return doCapi.SetDigitalOceanClusterProviderConfig(cluster.Spec.ClusterAPI, config)
	// add provider config to cluster
	//return cm.SetClusterProviderConfig()
}

// IsValid TODO: Add Description
func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	cfg := &api.SSHConfig{
		PrivateKey: SSHKey(cm.ctx).PrivateKey,
		User:       "root",
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

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, nil
}
