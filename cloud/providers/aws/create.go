package aws

import (
	"fmt"
	"net"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapi_aws "github.com/pharmer/pharmer/apis/v1beta1/aws"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (cm *ClusterManager) SetDefaultCluster() error {
	cluster := cm.Cluster
	n := namer{cluster: cluster}

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

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
		Cloud: api.CloudStatus{
			AWS: &api.AWSStatus{},
		},
	}
	cm.Cluster = cluster

	return cm.SetClusterProviderConfig()
}

func (cm *ClusterManager) SetClusterProviderConfig() error {
	conf := &clusterapi_aws.AWSClusterProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.AWSProviderGroupName + "/" + api.AWSProviderApiVersion,
			Kind:       api.AWSClusterProviderKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cm.Cluster.Name,
		},
		Region:     cm.Cluster.Spec.Config.Cloud.Region,
		SSHKeyName: cm.Cluster.Spec.Config.Cloud.SSHKeyName,
	}

	rawSpec, err := clusterapi_aws.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	return nil
}

/*func (cm *ClusterManager) SetupCerts() error {
	conf, err := clusterapi_aws.ClusterConfigFromProviderSpec(cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec)
	if err != nil {
		return err
	}

	conf.CAKeyPair = clusterapi_aws.KeyPair{
		Cert: cert.EncodeCertPEM(CACert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(CAKey(cm.ctx)),
	}
	conf.FrontProxyCAKeyPair = clusterapi_aws.KeyPair{
		Cert: cert.EncodeCertPEM(FrontProxyCACert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(FrontProxyCAKey(cm.ctx)),
	}
	conf.EtcdCAKeyPair = clusterapi_aws.KeyPair{
		Cert: cert.EncodeCertPEM(EtcdCaCert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(EtcdCaKey(cm.ctx)),
	}
	conf.SAKeyPair = clusterapi_aws.KeyPair{
		Cert: cert.EncodeCertPEM(SaCert(cm.ctx)),
		Key:  cert.EncodePrivateKeyPEM(SaKey(cm.ctx)),
	}

	rawSpec, err := clusterapi_aws.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	if _, err := store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
		return err
	}

	return nil
}*/

// IsValid TODO:add description
func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	cfg := &api.SSHConfig{
		PrivateKey: cm.Certs.SSHKey.PrivateKey,
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
