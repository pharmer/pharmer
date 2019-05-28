package gce

import (
	"net"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/google/uuid"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapiGCE "github.com/pharmer/pharmer/apis/v1beta1/gce"
	proconfig "github.com/pharmer/pharmer/apis/v1beta1/gce"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
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

// SetupCerts Loads necessary certs in Cluster Spec
func (cm *ClusterManager) SetupCerts() error {
	conf, err := clusterapiGCE.ClusterConfigFromProviderSpec(cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return err
	}

	conf.CAKeyPair = clusterapiGCE.KeyPair{
		Cert: cert.EncodeCertPEM(CACert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(CAKey(cm.ctx)),
	}
	conf.FrontProxyCAKeyPair = clusterapiGCE.KeyPair{
		Cert: cert.EncodeCertPEM(FrontProxyCACert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(FrontProxyCAKey(cm.ctx)),
	}
	conf.EtcdCAKeyPair = clusterapiGCE.KeyPair{
		Cert: cert.EncodeCertPEM(EtcdCaCert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(EtcdCaKey(cm.ctx)),
	}
	conf.SAKeyPair = clusterapiGCE.KeyPair{
		Cert: cert.EncodeCertPEM(SaCert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(SaKey(cm.ctx)),
	}

	rawSpec, err := clusterapiGCE.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	if _, err := Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
		return err
	}
	return nil
}

func (cm *ClusterManager) SetDefaultCluster(cluster *api.Cluster) error {
	n := namer{cluster: cluster}
	config := cluster.Spec.Config
	// Init object meta
	uid, _ := uuid.NewUUID()
	cluster.ObjectMeta.UID = types.UID(uid.String())
	cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster.ObjectMeta.Generation = time.Now().UnixNano()

	config.Cloud.InstanceImageProject = "ubuntu-os-cloud"
	config.Cloud.InstanceImage = "ubuntu-1604-xenial-v20170721"
	config.Cloud.OS = "ubuntu-1604-lts"
	config.Cloud.CCMCredentialName = config.CredentialName
	config.Cloud.GCE = &api.GoogleSpec{
		NetworkName: "default",
		NodeTags:    []string{n.NodePrefix()},
	}

	config.APIServerExtraArgs["cloud-config"] = "/etc/kubernetes/ccm/cloud-config"

	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)
	config.ControllerManagerExtraArgs = map[string]string{
		"cloud-config":   "/etc/kubernetes/ccm/cloud-config",
		"cloud-provider": config.Cloud.CloudProvider,
	}

	if cluster.Spec.ClusterAPI.ObjectMeta.Annotations == nil {
		cluster.Spec.ClusterAPI.ObjectMeta.Annotations = make(map[string]string)
	}

	spew.Dump(cluster)

	return clusterapiGCE.SetGCEclusterProviderConfig(cluster.Spec.ClusterAPI, config)
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

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, nil
}
