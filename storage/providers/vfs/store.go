package vfs

import (
	"context"
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
	storage.RegisterProvider(UID, func(ctx context.Context, cfg config.PharmerConfig) (storage.Interface, error) {
		return &FileStore{cfg: cfg}, nil
	})
}

type FileStore struct {
	cfg config.PharmerConfig
}

var _ storage.Interface = &FileStore{}

func (s *FileStore) Clusters() storage.ClusterStore {
	return s
}

func (s *FileStore) Instances() storage.InstanceStore {
	return s
}

func (s *FileStore) Credentials() storage.CredentialStore {
	return s
}

func (s *FileStore) Certificates() storage.CertificateStore {
	return s
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
func (s *FileStore) LoadInstance(name string) (*api.Instance, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FileStore) LoadInstances(cluster string) ([]*api.Instance, error) {
	return nil, errors.New("NotImplemented")
}

func (s *FileStore) SaveInstance(instance *api.Instance) error {
	return errors.New("NotImplemented")
}

func (s *FileStore) SaveInstances(instances []*api.Instance) error {
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
