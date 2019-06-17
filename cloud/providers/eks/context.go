package eks

import (
	"encoding/base64"
	"fmt"

	. "github.com/appscode/go/types"
	_eks "github.com/aws/aws-sdk-go/service/eks"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes" //"fmt"
	"k8s.io/client-go/rest"       //"gomodules.xyz/cert"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	*CloudManager
	conn *cloudConnector

	namer namer
}

var _ Interface = &ClusterManager{}

const (
	UID               = "eks"
	v1Prefix          = "k8s-aws-v1."
	clusterIDHeader   = "x-k8s-aws-id"
	EKSNodeConfigMap  = "aws-auth"
	EKSConfigMapRoles = "mapRoles"
	EKSVPCUrl         = "https://amazon-eks.s3-us-west-2.amazonaws.com/cloudformation/2019-01-09/amazon-eks-vpc-sample.yaml"
	ServiceRoleUrl    = "https://amazon-eks.s3-us-west-2.amazonaws.com/cloudformation/2019-01-09/amazon-eks-service-role.yaml"
	NodeGroupUrl      = "https://amazon-eks.s3-us-west-2.amazonaws.com/cloudformation/2019-01-09/amazon-eks-nodegroup.yaml"
)

func init() {
	RegisterCloudManager(UID, func(cluster *api.Cluster, certs *certificates.PharmerCertificates) Interface {
		return New(cluster, certs)
	})
}

func New(cluster *api.Cluster, certs *certificates.PharmerCertificates) Interface {
	return &ClusterManager{
		CloudManager: &CloudManager{
			Cluster: cluster,
			Certs:   certs,
		},
		namer: namer{
			cluster: cluster,
		},
	}
}

// AddToManager adds all Controllers to the Manager
func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	return nil
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return nil
}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	kc, err := cm.GetEKSAdminClient()
	if err != nil {
		return nil, err
	}
	return kc, nil
}

func (cm *ClusterManager) GetEKSAdminClient() (kubernetes.Interface, error) {
	resp, err := cm.conn.eks.DescribeCluster(&_eks.DescribeClusterInput{
		Name: StringP(cm.Cluster.Name),
	})
	if err != nil {
		return nil, err
	}

	token, err := cm.conn.getAuthenticationToken()
	if err != nil {
		return nil, err
	}

	caData, err := base64.StdEncoding.DecodeString(*resp.Cluster.CertificateAuthority.Data)
	if err != nil {
		return nil, err
	}

	cfg := &rest.Config{
		Host:        String(resp.Cluster.Endpoint),
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
	}
	return kubernetes.NewForConfig(cfg)

}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	cluster := cm.Cluster
	var err error
	cm.conn, err = NewConnector(cm)
	if err != nil {
		return nil, err
	}
	resp, err := cm.conn.eks.DescribeCluster(&_eks.DescribeClusterInput{
		Name: StringP(cluster.Name),
	})
	if err != nil {
		return nil, err
	}

	caData, err := base64.StdEncoding.DecodeString(*resp.Cluster.CertificateAuthority.Data)
	if err != nil {
		return nil, err
	}

	var (
		clusterName = fmt.Sprintf("%s.pharmer", cluster.Name)
		userName    = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
		ctxName     = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
	)
	cfg := api.KubeConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "KubeConfig",
		},
		Preferences: api.Preferences{
			Colors: true,
		},
		Cluster: api.NamedCluster{
			Name:                     clusterName,
			Server:                   String(resp.Cluster.Endpoint),
			CertificateAuthorityData: caData,
		},
		AuthInfo: api.NamedAuthInfo{
			Username: userName,
			Name:     userName,
			Exec: &api.ExecConfig{
				APIVersion: "client.authentication.k8s.io/v1alpha1",
				Command:    "guard",
				Args:       []string{"login", "-k", cluster.Name, "-p", "eks"},
			},
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	return nil
}

func (cm *ClusterManager) GetCloudConnector() error {
	conn, err := NewConnector(cm)
	cm.conn = conn
	return err
}

func (cm *ClusterManager) NewMasterTemplateData(machine *v1alpha1.Machine, token string, td TemplateData) TemplateData {
	return TemplateData{}
}

func (cm *ClusterManager) NewNodeTemplateData(machine *v1alpha1.Machine, token string, td TemplateData) TemplateData {
	return TemplateData{}
}

func (cm *ClusterManager) EnsureMaster() error {
	return nil
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	return ""
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return "", nil
}
