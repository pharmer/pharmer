package eks

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"

	. "github.com/appscode/go/types"
	_eks "github.com/aws/aws-sdk-go/service/eks"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes" //"fmt"
	"k8s.io/client-go/rest"       //"k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	// Deprecated
	namer namer
	m     sync.Mutex

	owner string
}

var _ Interface = &ClusterManager{}

const (
	UID               = "eks"
	RoleClusterUser   = "clusterUser"
	RoleClusterAdmin  = "clusterAdmin"
	v1Prefix          = "k8s-aws-v1."
	maxTokenLenBytes  = 1024 * 4
	clusterIDHeader   = "x-k8s-aws-id"
	EKSNodeConfigMap  = "aws-auth"
	EKSConfigMapRoles = "mapRoles"
	EKSVPCUrl         = "https://amazon-eks.s3-us-west-2.amazonaws.com/cloudformation/2019-01-09/amazon-eks-vpc-sample.yaml"
	ServiceRoleUrl    = "https://amazon-eks.s3-us-west-2.amazonaws.com/cloudformation/2019-01-09/amazon-eks-service-role.yaml"
	NodeGroupUrl      = "https://amazon-eks.s3-us-west-2.amazonaws.com/cloudformation/2019-01-09/amazon-eks-nodegroup.yaml"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
	return &ClusterManager{ctx: ctx}
}

// AddToManager adds all Controllers to the Manager
func (cm *ClusterManager) AddToManager(ctx context.Context, m manager.Manager) error {
	return nil
}

type paramK8sClient struct{}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return nil
}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	cm.m.Lock()
	defer cm.m.Unlock()

	v := cm.ctx.Value(paramK8sClient{})
	if kc, ok := v.(kubernetes.Interface); ok && kc != nil {
		return kc, nil
	}

	kc, err := cm.GetEKSAdminClient()
	if err != nil {
		return nil, err
	}
	cm.ctx = context.WithValue(cm.ctx, paramK8sClient{}, kc)
	return kc, nil
}

func (cm *ClusterManager) GetEKSAdminClient() (kubernetes.Interface, error) {
	resp, err := cm.conn.eks.DescribeCluster(&_eks.DescribeClusterInput{
		Name: StringP(cm.cluster.Name),
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

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	var err error
	cm.conn, err = NewConnector(cm.ctx, cluster, cm.owner)
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
