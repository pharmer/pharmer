package eks

import (
	"context"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
	//"fmt"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/util/cert"
	"encoding/base64"
	"fmt"
	"time"

	. "github.com/appscode/go/types"
	_eks "github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/sts"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//	clientset "k8s.io/client-go/kubernetes"
	//	"k8s.io/client-go/tools/clientcmd"
	//	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	// Deprecated
	namer namer
	m     sync.Mutex
}

var _ Interface = &ClusterManager{}

const (
	UID              = "eks"
	RoleClusterUser  = "clusterUser"
	RoleClusterAdmin = "clusterAdmin"
	v1Prefix         = "k8s-aws-v1."
	maxTokenLenBytes = 1024 * 4
	clusterIDHeader  = "x-k8s-aws-id"
	EKSNodeConfigMap = "aws-auth"
	EKSVPCUrl        = "https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-06-05/amazon-eks-vpc-sample.yaml"
	ServiceRoleUrl   = "https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-06-05/amazon-eks-service-role.yaml"
	NodeGroupUrl     = "https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-06-05/amazon-eks-nodegroup.yaml"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
	return &ClusterManager{ctx: ctx}
}

type paramK8sClient struct{}

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

	request, _ := cm.conn.sts.GetCallerIdentityRequest(&sts.GetCallerIdentityInput{})
	request.HTTPRequest.Header.Add(clusterIDHeader, cm.cluster.Name)
	// sign the request
	presignedURLString, err := request.Presign(60 * time.Second)
	token := v1Prefix + base64.RawURLEncoding.EncodeToString([]byte(presignedURLString))

	caData, err := base64.StdEncoding.DecodeString(*resp.Cluster.CertificateAuthority.Data)
	fmt.Println(err)

	cfg := &rest.Config{
		Host:        String(resp.Cluster.Endpoint),
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
	}
	return kubernetes.NewForConfig(cfg)

}
