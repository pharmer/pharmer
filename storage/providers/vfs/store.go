package vfs

import (
	"crypto/x509"
	"errors"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/storage"
)

const (
	UID = "vfs"
)

func init() {
	storage.RegisterStore(UID, func(cfg *config.PharmerConfig) (storage.Store, error) { return &FileStore{cfg: cfg}, nil })
}

type FileStore struct {
	cfg *config.PharmerConfig
}

var _ storage.Store = &FileStore{}

func (s *FileStore) Clusters() (storage.ClusterStore, bool) {
	return s, true
}

func (s *FileStore) Instances() (storage.InstanceStore, bool) {
	return s, true
}

func (s *FileStore) Credentials() (storage.CredentialStore, bool) {
	return s, true
}

func (s *FileStore) Certificates() (storage.CertificateStore, bool) {
	return s, true
}

// ClusterStore _______________________________________________________________
func (s *FileStore) GetActiveCluster(name string) ([]*api.Cluster, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FileStore) LoadCluster(name string) (*api.Cluster, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FileStore) SaveCluster(*api.Cluster) error {
	return errors.New("NotImplemented")
}

// InstanceStore ______________________________________________________________
func (s *FileStore) LoadInstance(name string) (*api.KubernetesInstance, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FileStore) LoadInstances(cluster string) ([]*api.KubernetesInstance, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FileStore) SaveInstance(instance *api.KubernetesInstance) error {
	return errors.New("NotImplemented")
}

func (s *FileStore) SaveInstances(instances []*api.KubernetesInstance) error {
	return errors.New("NotImplemented")
}

// CertificateStore ___________________________________________________________
func (s *FileStore) LoadCertificate(name string) (*x509.Certificate, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FileStore) SaveCertificate(*x509.Certificate) error {
	return errors.New("NotImplemented")
}

// CredentialStore ____________________________________________________________
