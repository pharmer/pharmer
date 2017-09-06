package fake

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"sync"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/store"
)

const (
	UID = "fake"
)

func init() {
	store.RegisterProvider(UID, func(ctx context.Context, cfg *api.PharmerConfig) (store.Interface, error) {
		return &FakeStore{
			instanceGroups: map[string]store.InstanceGroupStore{},
			instances:      map[string]store.InstanceStore{},
			certificates:   map[string]store.CertificateStore{},
			sshKeys:        map[string]store.SSHKeyStore{},
		}, nil
	})
}

type FakeStore struct {
	credentials    store.CredentialStore
	clusters       store.ClusterStore
	instanceGroups map[string]store.InstanceGroupStore
	instances      map[string]store.InstanceStore
	certificates   map[string]store.CertificateStore
	sshKeys        map[string]store.SSHKeyStore

	mux sync.Mutex
}

var _ store.Interface = &FakeStore{}

func (s *FakeStore) Credentials() store.CredentialStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.credentials == nil {
		s.credentials = &CredentialFileStore{container: map[string]*api.Credential{}}
	}
	return s.credentials
}

func (s *FakeStore) Clusters() store.ClusterStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.clusters == nil {
		s.clusters = &ClusterFileStore{container: map[string]*api.Cluster{}}
	}
	return s.clusters
}

func (s *FakeStore) InstanceGroups(cluster string) store.InstanceGroupStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.instanceGroups[cluster]; !found {
		s.instanceGroups[cluster] = &InstanceGroupFileStore{container: map[string]*api.InstanceGroup{}, cluster: cluster}
	}
	return s.instanceGroups[cluster]
}

func (s *FakeStore) Instances(cluster string) store.InstanceStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.instances[cluster]; !found {
		s.instances[cluster] = &InstanceFileStore{container: map[string]*api.Instance{}, cluster: cluster}
	}
	return s.instances[cluster]
}

func (s *FakeStore) Certificates(cluster string) store.CertificateStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.certificates[cluster]; !found {
		s.certificates[cluster] = &CertificateFileStore{certs: map[string]*x509.Certificate{}, keys: map[string]*rsa.PrivateKey{}, cluster: cluster}
	}
	return s.certificates[cluster]
}

func (s *FakeStore) SSHKeys(cluster string) store.SSHKeyStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.sshKeys[cluster]; !found {
		s.sshKeys[cluster] = &SSHKeyFileStore{container: map[string][]byte{}, cluster: cluster}
	}
	return s.sshKeys[cluster]
}
