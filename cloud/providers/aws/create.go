package aws

import (
	"fmt"
	"net"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapi_aws "github.com/pharmer/pharmer/apis/v1beta1/aws"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func (cm *ClusterManager) GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error) {
	cluster := cm.Cluster
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
	cluster.Spec.Config.APIServerExtraArgs = map[string]string{
		"cloud-provider": cluster.Spec.Config.Cloud.CloudProvider,
	}

	// Init status
	cluster.Status = api.PharmerClusterStatus{
		Phase: api.ClusterPending,
		Cloud: api.CloudStatus{
			AWS: &api.AWSStatus{},
		},
	}

	return cm.SetClusterProviderConfig()
}

func (cm *ClusterManager) SetClusterProviderConfig() error {
	certs := cm.Certs
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

		CAKeyPair: clusterapi_aws.KeyPair{
			Cert: cert.EncodeCertPEM(certs.CACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(certs.CACert.Key),
		},
		EtcdCAKeyPair: clusterapi_aws.KeyPair{
			Cert: cert.EncodeCertPEM(certs.EtcdCACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(certs.EtcdCACert.Key),
		},
		FrontProxyCAKeyPair: clusterapi_aws.KeyPair{
			Cert: cert.EncodeCertPEM(certs.FrontProxyCACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(certs.FrontProxyCACert.Key),
		},
		SAKeyPair: clusterapi_aws.KeyPair{
			Cert: cert.EncodeCertPEM(certs.ServiceAccountCert.Cert),
			Key:  cert.EncodePrivateKeyPEM(certs.ServiceAccountCert.Key),
		},
	}

	rawSpec, err := clusterapi_aws.EncodeClusterSpec(conf)
	if err != nil {
		return err
	}

	cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec.Value = rawSpec

	return nil
}

func (cm *ClusterManager) GetSSHConfig(node *core.Node) (*api.SSHConfig, error) {
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
		return nil, errors.Errorf("failed to detect external Ip for node %s of cluster %s", node.Name, cm.Cluster.Name)
	}
	return cfg, nil
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	return kube.GetAdminConfig(cm.Cluster, cm.GetCaCertPair())
}

func (cm *ClusterManager) SetDefaults(cluster *api.Cluster) error {
	return errors.New("not implemented")
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return errors.New("Not Implemented")
}
