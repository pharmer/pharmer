package aws

import (
	"fmt"
	"net"
	"strings"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapi_aws "github.com/pharmer/pharmer/apis/v1beta1/aws"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(cluster *api.Cluster, sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	spec := clusterapi_aws.AWSMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.AWSProviderGroupName + "/" + api.AWSProviderApiVersion,
			Kind:       api.AWSMachineProviderKind,
		},
		InstanceType: sku,
		KeyName:      cluster.Spec.Config.Cloud.SSHKeyName,
	}

	if role == "Node" {
		spec.IAMInstanceProfile = cluster.Spec.Config.Cloud.AWS.IAMProfileNode
	} else {
		spec.IAMInstanceProfile = cluster.Spec.Config.Cloud.AWS.IAMProfileMaster
	}

	providerSpec, err := clusterapi_aws.EncodeMachineSpec(&spec)
	if err != nil {
		return clusterapi.ProviderSpec{}, err
	}

	return clusterapi.ProviderSpec{
		Value: providerSpec,
	}, nil
}

// SetOwner sets owner field of ClusterManager
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

	cluster.SetNetworkingDefaults(config.Cloud.NetworkProvider)

	config.APIServerCertSANs = NameGenerator(cm.ctx).ExtraNames(cluster.Name)
	config.APIServerExtraArgs = map[string]string{
		// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
		"kubelet-preferred-address-types": strings.Join([]string{
			string(core.NodeInternalIP),
			string(core.NodeInternalDNS),
			string(core.NodeExternalDNS),
			string(core.NodeExternalIP),
		}, ","),
		"cloud-provider": cluster.Spec.Config.Cloud.CloudProvider,
	}

	// Init spec
	cluster.Spec.Config.Cloud.Region = cluster.Spec.Config.Cloud.Zone[0 : len(cluster.Spec.Config.Cloud.Zone)-1]
	cluster.Spec.Config.Cloud.SSHKeyName = n.GenSSHKeyExternalID()
	if cluster.Spec.Config.Cloud.AWS == nil {
		cluster.Spec.Config.Cloud.AWS = &api.AWSSpec{}
	}
	cluster.Spec.Config.Cloud.AWS.MasterSGName = n.GenMasterSGName()
	cluster.Spec.Config.Cloud.AWS.NodeSGName = n.GenNodeSGName()
	cluster.Spec.Config.Cloud.AWS.BastionSGName = n.GenBastionSGName()

	cluster.Spec.Config.Cloud.AWS.IAMProfileMaster = fmt.Sprintf("master.%v.pharmer", cluster.Name)
	cluster.Spec.Config.Cloud.AWS.IAMProfileNode = fmt.Sprintf("node.%v.pharmer", cluster.Name)
	cluster.Spec.Config.Cloud.AWS.VpcCIDRBase = "10.0"
	cluster.Spec.Config.Cloud.AWS.MasterIPSuffix = ".9"
	cluster.Spec.Config.Cloud.AWS.VpcCIDR = "10.0.0.0/16"
	cluster.Spec.Config.Cloud.AWS.PublicSubnetCIDR = "10.0.1.0/24"
	cluster.Spec.Config.Cloud.AWS.PrivateSubnetCIDR = "10.0.0.0/24"

	if cluster.IsMinorVersion("1.9") {
		config.APIServerExtraArgs["admission-control"] = api.DeprecatedV19AdmissionControl
	}
	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
		Cloud: api.CloudStatus{
			AWS: &api.AWSStatus{},
		},
	}
	cm.cluster = cluster

	return cm.SetClusterProviderConfig()
}

func (cm *ClusterManager) SetClusterProviderConfig() error {
	conf := &clusterapi_aws.AWSClusterProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.AWSProviderGroupName + "/" + api.AWSProviderApiVersion,
			Kind:       api.AWSClusterProviderKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cm.cluster.Name,
		},
		Region:     cm.cluster.Spec.Config.Cloud.Region,
		SSHKeyName: cm.cluster.Spec.Config.Cloud.SSHKeyName,
	}

	rawSpec, err := clusterapi_aws.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	return nil
}

func (cm *ClusterManager) SetupCerts() error {
	caCert, caKey, err := Store(cm.ctx).Owner(cm.owner).Certificates(cm.cluster.Name).Get(cm.cluster.Spec.Config.CACertName)
	if err != nil {
		return err
	}
	fpCert, fpKey, err := Store(cm.ctx).Owner(cm.owner).Certificates(cm.cluster.Name).Get(cm.cluster.Spec.Config.FrontProxyCACertName)
	if err != nil {
		return err
	}

	conf, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return err
	}

	conf.CAKeyPair = clusterapi_aws.KeyPair{
		Cert: cert.EncodeCertPEM(caCert),
		Key:  cert.EncodePrivateKeyPEM(caKey),
	}
	conf.FrontProxyCAKeyPair = clusterapi_aws.KeyPair{
		Cert: cert.EncodeCertPEM(fpCert),
		Key:  cert.EncodePrivateKeyPEM(fpKey),
	}

	rawSpec, err := clusterapi_aws.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cm.cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	if _, err := Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
		return err
	}

	return nil
}

// IsValid TODO:add description
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

func (cm *ClusterManager) SetDefaults(cluster *api.Cluster) error {
	return errors.New("not implemented")
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return errors.New("Not Implemented")
}
