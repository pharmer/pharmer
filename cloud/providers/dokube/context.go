package dokube

import (
	"context"
	"fmt"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
	m       sync.Mutex

	owner string
}

var _ Interface = &ClusterManager{}

const (
	UID = "dokube"
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

	kc, err := NewDokubeAdminClient(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return nil, err
	}
	cm.ctx = context.WithValue(cm.ctx, paramK8sClient{}, kc)
	return kc, nil
}

func NewDokubeAdminClient(ctx context.Context, cluster *api.Cluster, owner string) (kubernetes.Interface, error) {
	adminCert, adminKey, err := GetAdminCertificate(ctx, cluster, owner)
	if err != nil {
		return nil, err
	}
	host := cluster.APIServerURL()
	if host == "" {
		return nil, errors.Errorf("failed to detect api server url for cluster %s", cluster.Name)
	}
	cfg := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(CACert(ctx)),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}

	return kubernetes.NewForConfig(cfg)
}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	var err error
	cm.ctx, err = LoadCACertificates(cm.ctx, cluster, cm.owner)
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
			Server:                   cluster.APIServerURL(),
			CertificateAuthorityData: cert.EncodeCertPEM(CACert(cm.ctx)),
		},
		AuthInfo: api.NamedAuthInfo{
			Name: userName,
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}

/*func (cm *ClusterManager) kubeConfig(cluster *api.Cluster) (*api.KubeConfig, error){
	var err error
	cm.conn, err = NewConnector(cm.ctx, cluster, cm.owner)
	kcc, _, err := cm.conn.client.Kubernetes.GetKubeConfig(cm.ctx, cluster.Spec.Config.Cloud.Dokube.ClusterID)
	fmt.Println(err)
	if err != nil {
		return nil, err
	}

	var kc api.KubeConfig
	err = yaml.Unmarshal(kcc.KubeconfigYAML, &kc)
	fmt.Println(err)
	if err != nil {
		return nil, err
	}
	spew.Dump(kc)
	return &kc, nil
}*/