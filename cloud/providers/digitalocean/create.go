package digitalocean

import (
	"encoding/json"
	"net"
	"strings"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
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
	spec := &api.DigitalOceanMachineProviderConfig{
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
	//cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	config.Cloud.InstanceImage = "ubuntu-16-04-x64"

	//cluster.InitializeClusterApi()
	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)

	//kube.Spec.AuthorizationModes = strings.Split(kubeadmapi.DefaultAuthorizationModes, ",")
	config.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	config.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalIP),
			string(core.NodeExternalIP),
		}, ","),
		//	"endpoint-reconciler-type": "lease",
	}

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}

	// add provider config to cluster
	return cluster.SetDigitalOceanProviderConfig(cluster.Spec.ClusterAPI, config)
}

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
