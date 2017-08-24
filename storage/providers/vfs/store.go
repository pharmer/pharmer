package vfs

import (
	"context"

	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/storage"
)

const (
	UID      = "vfs"
	pageSize = 50
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

func (s *FileStore) Credentials() storage.CredentialStore {
	return &CredentialFileStore{}
}

func (s *FileStore) Clusters() storage.ClusterStore {
	return &ClusterFileStore{}
}

func (s *FileStore) Instances(name string) storage.InstanceStore {
	return &InstanceFileStore{cluster: name}
}

func (s *FileStore) Certificates(name string) storage.CertificateStore {
	return &CertificateFileStore{cluster: name}
}

func (s *FileStore) SSHKeys(name string) storage.SSHKeyStore {
	return &SSHKeyFileStore{cluster: name}
}
