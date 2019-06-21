package cloud

import (
	"github.com/go-logr/logr"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/klogr"
)

type CloudManagerInterface interface {
	GetCluster() *api.Cluster
	GetCaCertPair() *certificates.CertKeyPair
	GetCertificates() *certificates.Certificates
	GetCredential() (*cloudapi.Credential, error)

	GetAdminClient() (kubernetes.Interface, error)
	GetCloudManager() (Interface, error)
}

var _ CloudManagerInterface = &Scope{}

type Scope struct {
	Cluster        *api.Cluster
	Certs          *certificates.Certificates
	CredentialData map[string]string
	StoreProvider  store.ResourceInterface
	CloudManager   Interface
	AdminClient    kubernetes.Interface
	logr.Logger
}

type NewScopeParams struct {
	Cluster       *api.Cluster
	Certs         *certificates.Certificates
	StoreProvider store.ResourceInterface
	Logger        logr.Logger
}

func NewScope(params NewScopeParams) *Scope {
	if params.Logger == nil {
		params.Logger = klogr.New().WithValues("cluster-name", params.Cluster.Name)
	}
	return &Scope{
		Cluster:       params.Cluster,
		Certs:         params.Certs,
		StoreProvider: params.StoreProvider,
		Logger:        params.Logger,
	}
}

func (s *Scope) GetCredential() (*cloudapi.Credential, error) {
	return s.StoreProvider.Credentials().Get(s.Cluster.Spec.Config.CredentialName)
}

func (s *Scope) GetCluster() *api.Cluster {
	return s.Cluster
}

func (s *Scope) GetCloudManager() (Interface, error) {
	if s.CloudManager != nil {
		return s.CloudManager, nil
	}
	var err error
	s.CloudManager, err = GetCloudManager(s)
	return s.CloudManager, err
}

func (s *Scope) GetCaCertPair() *certificates.CertKeyPair {
	return &s.Certs.CACert
}

func (s *Scope) GetCertificates() *certificates.Certificates {
	return s.Certs
}

func (s *Scope) GetAdminClient() (kubernetes.Interface, error) {
	if s.AdminClient != nil {
		return s.AdminClient, nil
	}

	client, err := kube.NewAdminClient(&s.Certs.CACert, s.Cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new kube-client")
	}
	s.AdminClient = client

	return client, nil
}
