package gke

import (
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	defaultNetwork = "default"
)

type ClusterManager struct {
	*CloudManager

	conn *cloudConnector
}

const (
	UID = "gke"
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
	}
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	return nil
}

func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	return nil
}

func (cm *ClusterManager) GetCloudConnector() error {
	if cm.conn != nil {
		return nil
	}
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

var _ Interface = &ClusterManager{}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return nil
}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	kc, err := cm.NewGKEAdminClient()
	if err != nil {
		return nil, err
	}
	return kc, nil
}

func (cm *ClusterManager) NewGKEAdminClient() (kubernetes.Interface, error) {
	cluster := cm.Cluster
	adminCert, adminKey, err := certificates.GetAdminCertificate(cm.Cluster.Name)
	if err != nil {
		return nil, err
	}
	host := cluster.APIServerURL()
	if host == "" {
		return nil, errors.Errorf("failed to detect api server url for cluster %s", cluster.Name)
	}
	cfg := &rest.Config{
		Host:     host,
		Username: cluster.Spec.Config.Cloud.GKE.UserName,
		Password: cluster.Spec.Config.Cloud.GKE.Password,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(cm.GetCaCertPair().Cert),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}

	return kubernetes.NewForConfig(cfg)
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	cluster := cm.Cluster

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
			CertificateAuthorityData: cert.EncodeCertPEM(cm.GetCaCertPair().Cert),
		},
		AuthInfo: api.NamedAuthInfo{
			Name:     userName,
			Username: cluster.Spec.Config.Cloud.GKE.UserName,
			Password: cluster.Spec.Config.Cloud.GKE.Password,
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}
