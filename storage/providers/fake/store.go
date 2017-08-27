package fake

import (
	"context"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/storage"
)

const (
	UID      = "fake"
	pageSize = 50
)

func init() {
	storage.RegisterProvider(UID, func(ctx context.Context, cfg *api.PharmerConfig) (storage.Interface, error) {
		return &FakeStore{}, nil
	})
}

type FakeStore struct{}

var _ storage.Interface = &FakeStore{}

func (s *FakeStore) Credentials() storage.CredentialStore {
	return &CredentialFileStore{container: map[string]*api.Credential{}}
}

func (s *FakeStore) Clusters() storage.ClusterStore {
	return &ClusterFileStore{container: map[string]*api.Cluster{}}
}

func (s *FakeStore) Instances(name string) storage.InstanceStore {
	return &InstanceFileStore{container: map[string]*api.Instance{}, cluster: name}
}

func (s *FakeStore) Certificates(name string) storage.CertificateStore {
	return &CertificateFileStore{container: map[string][]byte{}, cluster: name}
}

func (s *FakeStore) SSHKeys(name string) storage.SSHKeyStore {
	return &SSHKeyFileStore{container: map[string][]byte{}, cluster: name}
}
