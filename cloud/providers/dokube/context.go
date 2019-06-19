package dokube

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	*cloud.Scope

	conn *cloudConnector
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "dokube"
)

func init() {
	cloud.RegisterCloudManager(UID, New)
}

func New(s *cloud.Scope) cloud.Interface {
	return &ClusterManager{
		Scope: s,
	}
}

func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	return nil
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	return nil
}

func (cm *ClusterManager) SetCloudConnector() error {
	conn, err := newconnector(cm)
	cm.conn = conn
	return err
}

func (cm *ClusterManager) NewMasterTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	return cloud.TemplateData{}
}

func (cm *ClusterManager) NewNodeTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	return cloud.TemplateData{}
}

func (cm *ClusterManager) EnsureMaster(_ *v1alpha1.Machine) error {
	return nil
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	return ""
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return "", nil
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return nil
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	return kube.GetAdminConfig(cm.Cluster, cm.GetCaCertPair())
}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	kc, err := NewDokubeAdminClient(cm)
	if err != nil {
		return nil, err
	}
	return kc, nil
}

func NewDokubeAdminClient(cm *ClusterManager) (kubernetes.Interface, error) {
	adminCert, adminKey, err := certificates.GetAdminCertificate(cm.StoreProvider.Certificates(cm.Cluster.Name))
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
