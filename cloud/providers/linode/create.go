package linode

import (
	"net"
	"strings"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func (cm *ClusterManager) GetDefaultNodeSpec(cluster *api.Cluster, sku string) (api.NodeSpec, error) {
	if sku == "" {
		sku = "4"
	}
	return api.NodeSpec{
		SKU: sku,
		//DiskType:      "",
		//DiskSize:      100,
	}, nil
}

func (cm *ClusterManager) SetDefaultCluster(cluster *api.Cluster, config *api.ClusterProviderConfig) error {
	n := namer{cluster: cluster}

	// Init object meta
	if err := api.AssignTypeKind(cluster); err != nil {
		return err
	}

	// Init spec
	config.Cloud.Region = config.Cloud.Zone
	config.Cloud.SSHKeyName = n.GenSSHKeyExternalID()
	config.API.BindPort = kubeadmapi.DefaultAPIBindPort
	config.Cloud.InstanceImage = "ubuntu-16-04-x64"
	config.ETCDServers = []string{}

	cluster.InitializeClusterApi()
	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)

	config.AuthorizationModes = strings.Split(kubeadmapi.DefaultAuthorizationModes, ",")
	config.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	config.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalIP),
			string(core.NodeExternalIP),
		}, ","),
	}
	if config.IsMinorVersion("1.9") {
		config.APIServerExtraArgs["admission-control"] = api.DefaultV19AdmissionControl
	}

	config.Cloud.CCMCredentialName = config.CredentialName
	config.Cloud.Linode = &api.LinodeSpec{
		RootPassword: rand.GeneratePassword(),
	}

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
	}

	// add provider config to cluster
	cluster.SetProviderConfig(config)

	return nil
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
