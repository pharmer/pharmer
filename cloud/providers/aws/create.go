package aws

import (
	"fmt"
	"net"
	"strings"
	"time"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha3"
)

func (cm *ClusterManager) GetDefaultNodeSpec(cluster *api.Cluster, sku string) (api.NodeSpec, error) {
	if sku == "" {
		// assign at the time of apply
	}
	return api.NodeSpec{
		SKU:      sku,
		DiskType: "gp2",
		DiskSize: 100,
	}, nil
}

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) SetDefaults(cluster *api.Cluster) error {
	n := namer{cluster: cluster}

	// Init object meta
	cluster.ObjectMeta.UID = uuid.NewUUID()
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()
	api.AssignTypeKind(cluster)

	// Init spec
	cluster.Spec.Cloud.Region = cluster.Spec.Cloud.Zone[0 : len(cluster.Spec.Cloud.Zone)-1]
	cluster.Spec.Cloud.SSHKeyName = n.GenSSHKeyExternalID()
	if cluster.Spec.Cloud.AWS == nil {
		cluster.Spec.Cloud.AWS = &api.AWSSpec{}
	}
	cluster.Spec.Cloud.AWS.MasterSGName = n.GenMasterSGName()
	cluster.Spec.Cloud.AWS.NodeSGName = n.GenNodeSGName()

	cluster.Spec.Cloud.AWS.IAMProfileMaster = fmt.Sprintf("master.%v.pharmer", cluster.Name)
	cluster.Spec.Cloud.AWS.IAMProfileNode = fmt.Sprintf("node.%v.pharmer", cluster.Name)
	cluster.Spec.Cloud.AWS.VpcCIDRBase = "172.20"
	cluster.Spec.Cloud.AWS.MasterIPSuffix = ".9"
	cluster.Spec.Cloud.AWS.VpcCIDR = "172.20.0.0/16"
	cluster.Spec.Cloud.AWS.SubnetCIDR = "172.20.0.0/24"
	cluster.Spec.Networking.SetDefaults()
	cluster.Spec.Networking.MasterSubnet = "10.246.0.0/24"
	cluster.Spec.Networking.NonMasqueradeCIDR = "10.0.0.0/8"
	cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	cluster.Spec.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	cluster.Spec.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalDNS),
			string(core.NodeInternalIP),
			string(core.NodeExternalDNS),
			string(core.NodeExternalIP),
		}, ","),
		"cloud-provider": cluster.Spec.Cloud.CloudProvider,
	}
	if cluster.IsMinorVersion("1.9") {
		cluster.Spec.APIServerExtraArgs["admission-control"] = api.DeprecatedV19AdmissionControl
	}
	// Init status
	cluster.Status = api.ClusterStatus{
		Phase: api.ClusterPending,
		Cloud: api.CloudStatus{
			AWS: &api.AWSStatus{},
		},
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
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, nil
}
