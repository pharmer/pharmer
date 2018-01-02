package gce

import (
	"fmt"
	"net"
	"strings"
	"time"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

func (cm *ClusterManager) GetDefaultNodeSpec(cluster *api.Cluster, sku string) (api.NodeSpec, error) {
	if sku == "" {
		// assign at the time of apply
	}
	return api.NodeSpec{
		SKU:      sku,
		DiskType: "pd-standard",
		DiskSize: 100,
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
	cluster.Spec.Cloud.Region = cluster.Spec.Cloud.Zone[0:strings.LastIndex(cluster.Spec.Cloud.Zone, "-")]
	cluster.Spec.Cloud.SSHKeyName = n.GenSSHKeyExternalID()
	cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	// REGISTER_MASTER_KUBELET = false // always false, keep master lightweight
	// PREEMPTIBLE_NODE = false // Removed Support

	cluster.Spec.Cloud.InstanceImageProject = "ubuntu-os-cloud"
	cluster.Spec.Cloud.InstanceImage = "ubuntu-1604-xenial-v20170721"
	cluster.Spec.Cloud.CCMCredentialName = cluster.Spec.CredentialName
	cluster.Spec.Cloud.GCE = &api.GoogleSpec{
		NetworkName: "default",
		NodeTags:    []string{n.NodePrefix()},
	}
	cluster.Spec.Networking.NonMasqueradeCIDR = "10.0.0.0/8"
	cluster.Spec.Networking.PodSubnet = "10.244.0.0/16"
	cluster.Spec.AuthorizationModes = strings.Split(kubeadmapi.DefaultAuthorizationModes, ",")
	cluster.Spec.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	cluster.Spec.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeHostName),
			string(core.NodeInternalDNS),
			string(core.NodeInternalIP),
			string(core.NodeExternalDNS),
			string(core.NodeExternalIP),
		}, ","),
		"cloud-config": "/etc/kubernetes/ccm/cloud-config",
	}
	if cluster.IsMinorVersion("1.9") {
		cluster.Spec.APIServerExtraArgs["admission-control"] = api.DeprecatedV19AdmissionControl
	}
	cluster.Spec.ControllerManagerExtraArgs = map[string]string{
		"cloud-config": "/etc/kubernetes/ccm/cloud-config",
	}

	// Init status
	cluster.Status = api.ClusterStatus{
		Phase: api.ClusterPending,
	}
	return nil
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
		return nil, fmt.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cluster.Name)
	}
	return cfg, nil
}
