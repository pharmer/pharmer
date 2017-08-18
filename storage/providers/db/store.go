package db

import (
	"crypto/x509"
	"errors"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/storage"
)

const (
	UID = "db"
)

func init() {
	storage.RegisterStore(UID, func(cfg *config.PharmerConfig) (storage.Store, error) { return &SqlStore{cfg: cfg}, nil })
}

type SqlStore struct {
	cfg *config.PharmerConfig
}

var _ storage.Store = &SqlStore{}

func (s *SqlStore) Clusters() storage.ClusterStore {
	return s
}

func (s *SqlStore) Instances() storage.InstanceStore {
	return s
}

func (s *SqlStore) Credentials() storage.CredentialStore {
	return s
}

func (s *SqlStore) Certificates() storage.CertificateStore {
	return s
}

// ClusterStore _______________________________________________________________
func (s *SqlStore) GetActiveCluster(name string) ([]*api.Cluster, error) {
	return nil, errors.New("NotImplemented")
}

func (s *SqlStore) LoadCluster(name string) (*api.Cluster, error) {
	return nil, errors.New("NotImplemented")
}

func (s *SqlStore) SaveCluster(*api.Cluster) error {
	return errors.New("NotImplemented")
}

// InstanceStore ______________________________________________________________
func (s *SqlStore) LoadInstance(name string) (*api.Instance, error) {
	return nil, errors.New("NotImplemented")
}

func (s *SqlStore) LoadInstances(cluster string) ([]*api.Instance, error) {
	return nil, errors.New("NotImplemented")
}

func (s *SqlStore) SaveInstance(instance *api.Instance) error {
	return errors.New("NotImplemented")
}

func (s *SqlStore) SaveInstances(instances []*api.Instance) error {
	return errors.New("NotImplemented")
}

// CertificateStore ___________________________________________________________
func (s *SqlStore) LoadCertificate(name string) (*x509.Certificate, error) {
	return nil, errors.New("NotImplemented")
}

func (s *SqlStore) SaveCertificate(*x509.Certificate) error {
	return errors.New("NotImplemented")
}

// CredentialStore ____________________________________________________________
