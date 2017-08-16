package fake

import (
	"crypto/x509"
	"errors"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/storage"
)

const (
	UID = "fake"
)

func init() {
	storage.RegisterStore(UID, func(cfg *config.PharmerConfig) (storage.Store, error) { return &FakeStore{Config: cfg}, nil })
}

type FakeStore struct {
	Config *config.PharmerConfig
}

var _ storage.Store = &FakeStore{}

func (s *FakeStore) Clusters() (storage.ClusterStore, bool) {
	return s, true
}

func (s *FakeStore) Instances() (storage.InstanceStore, bool) {
	return s, true
}

func (s *FakeStore) Credentials() (storage.CredentialStore, bool) {
	return s, true
}

func (s *FakeStore) Certificates() (storage.CertificateStore, bool) {
	return s, true
}

// ClusterStore _______________________________________________________________
func (s *FakeStore) GetActiveCluster(name string) ([]*api.Cluster, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FakeStore) LoadCluster(name string) (*api.Cluster, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FakeStore) SaveCluster(*api.Cluster) error {
	return errors.New("NotImplemented")
}

// InstanceStore ______________________________________________________________
func (s *FakeStore) LoadInstance(name string) (*api.KubernetesInstance, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FakeStore) LoadInstances(cluster string) ([]*api.KubernetesInstance, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FakeStore) SaveInstance(instance *api.KubernetesInstance) error {
	return errors.New("NotImplemented")
}

func (s *FakeStore) SaveInstances(instances []*api.KubernetesInstance) error {
	return errors.New("NotImplemented")
}

// CertificateStore ___________________________________________________________
func (s *FakeStore) LoadCertificate(name string) (*x509.Certificate, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FakeStore) SaveCertificate(*x509.Certificate) error {
	return errors.New("NotImplemented")
}

// CredentialStore ____________________________________________________________
