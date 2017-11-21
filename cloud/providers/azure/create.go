package azure

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/appscode/go/crypto/rand"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func (cm *ClusterManager) GetDefaultNodeSpec(sku string) (api.NodeSpec, error) {
	if sku == "" {
		sku = "Standard_D2_v2"
	}
	return api.NodeSpec{
		SKU: sku,
		//	DiskType:      "",
		//	DiskSize:      100,
	}, nil
}

func (cm *ClusterManager) SetDefaults(cluster *api.Cluster) error {
	n := namer{cluster: cluster}

	// Init object meta
	cluster.ObjectMeta.UID = uuid.NewUUID()
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()
	api.AssignTypeKind(cluster)

	// Init spec
	cluster.Spec.Cloud.Region = cluster.Spec.Cloud.Zone
	cluster.Spec.Networking.NonMasqueradeCIDR = "10.0.0.0/8"
	cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	cluster.Spec.AuthorizationModes = strings.Split(kubeadmapi.DefaultAuthorizationModes, ",")
	cluster.Spec.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	cluster.Spec.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalDNS),
			string(core.NodeInternalIP),
			string(core.NodeExternalDNS),
			string(core.NodeExternalIP),
		}, ","),
		"cloud-config": "/etc/kubernetes/ccm/cloud-config",
	}
	cluster.Spec.ControllerManagerExtraArgs = map[string]string{
		"cloud-config": "/etc/kubernetes/ccm/cloud-config",
	}

	cluster.Spec.Cloud.CCMCredentialName = cluster.Spec.CredentialName
	cluster.Spec.Cloud.Azure = &api.AzureSpec{
		ResourceGroup:      n.ResourceGroupName(),
		SubnetName:         n.SubnetName(),
		SecurityGroupName:  n.NetworkSecurityGroupName(),
		VnetName:           n.VirtualNetworkName(),
		RouteTableName:     n.RouteTableName(),
		StorageAccountName: n.GenStorageAccountName(),
		SubnetCIDR:         "10.240.0.0/16",
		RootPassword:       rand.GeneratePassword(),
	}

	// Init status
	cluster.Status = api.ClusterStatus{
		Phase:            api.ClusterPending,
		SSHKeyExternalID: n.GenSSHKeyExternalID(),
	}

	return nil
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	cfg := &api.SSHConfig{
		PrivateKey: SSHKey(cm.ctx).PrivateKey,
		User:       "ubuntu",
		HostPort:   int32(22),
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == core.NodeExternalIP {
			cfg.HostIP = addr.Address
		}
	}
	if net.ParseIP(cfg.HostIP) == nil {
		return nil, fmt.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}
