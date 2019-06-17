package dokube

import (
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	*cloud.CloudManager

	conn *cloudConnector
}

func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	return nil
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	return nil
}

func (cm *ClusterManager) GetCloudConnector() error {
	conn, err := NewConnector(cm)
	cm.conn = conn
	return err
}

func (cm *ClusterManager) NewMasterTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	return cloud.TemplateData{}
}

func (cm *ClusterManager) NewNodeTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	return cloud.TemplateData{}
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

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "dokube"
)

func init() {
	cloud.RegisterCloudManager(UID, New)
}

func New(cluster *api.Cluster, certs *certificates.Certificates) cloud.Interface {
	return &ClusterManager{
		CloudManager: &cloud.CloudManager{
			Cluster: cluster,
			Certs:   certs,
		},
	}
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return nil
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	adminCert, adminKey, err := store.StoreProvider.Certificates(cm.Cluster.Name).Get("admin")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get admin cert and key")
	}

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
			CertificateAuthorityData: cert.EncodeCertPEM(cm.Certs.CACert.Cert),
		},
		AuthInfo: api.NamedAuthInfo{
			Name:                  userName,
			ClientCertificateData: cert.EncodeCertPEM(adminCert),
			ClientKeyData:         cert.EncodePrivateKeyPEM(adminKey),
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	kc, err := NewDokubeAdminClient(cm)
	if err != nil {
		return nil, err
	}
	return kc, nil
}

func NewDokubeAdminClient(cm *ClusterManager) (kubernetes.Interface, error) {
	adminCert, adminKey, err := certificates.GetAdminCertificate(cm.Cluster.Name)
	if err != nil {
		return nil, err
	}
	host := cm.Cluster.APIServerURL()
	if host == "" {
		return nil, errors.Errorf("failed to detect api server url for cluster %s", cm.Cluster.Name)
	}
	cfg := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(cm.Certs.CACert.Cert),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}

	return kubernetes.NewForConfig(cfg)
}
