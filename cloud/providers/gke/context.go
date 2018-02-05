package gke

import (
	"context"
	"fmt"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
)

const (
	maxInstancesPerMIG = 5 // Should be 500
	defaultNetwork     = "default"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
	m       sync.Mutex
}

var _ Interface = &ClusterManager{}

const (
	UID = "gke"
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

	kc, err := NewGKEAdminClient(cm.ctx, cm.cluster)
	if err != nil {
		return nil, err
	}
	cm.ctx = context.WithValue(cm.ctx, paramK8sClient{}, kc)
	return kc, nil
}

func NewGKEAdminClient(ctx context.Context, cluster *api.Cluster) (kubernetes.Interface, error) {
	adminCert, adminKey, err := GetAdminCertificate(ctx, cluster)
	if err != nil {
		return nil, err
	}
	host := cluster.APIServerURL()
	if host == "" {
		return nil, fmt.Errorf("failed to detect api server url for cluster %s", cluster.Name)
	}
	cfg := &rest.Config{
		Host:     host,
		Username: cluster.Spec.Cloud.GKE.UserName,
		Password: cluster.Spec.Cloud.GKE.Password,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(CACert(ctx)),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}

	return kubernetes.NewForConfig(cfg)
}
