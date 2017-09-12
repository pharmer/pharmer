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
			nodeGroups:   map[string]store.NodeGroupStore{},
			nodes:        map[string]store.InstanceStore{},
			certificates: map[string]store.CertificateStore{},
			sshKeys:      map[string]store.SSHKeyStore{},
		}, nil
	})
}

type FakeStore struct {
	credentials  store.CredentialStore
	clusters     store.ClusterStore
	nodeGroups   map[string]store.NodeGroupStore
	nodes        map[string]store.InstanceStore
	certificates map[string]store.CertificateStore
	sshKeys      map[string]store.SSHKeyStore

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

func (s *FakeStore) NodeGroups(cluster string) store.NodeGroupStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.nodeGroups[cluster]; !found {
		s.nodeGroups[cluster] = &NodeGroupFileStore{container: map[string]*api.NodeGroup{}, cluster: cluster}
	}
	return s.nodeGroups[cluster]
}

func (s *FakeStore) Instances(cluster string) store.InstanceStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.nodes[cluster]; !found {
		s.nodes[cluster] = &InstanceFileStore{container: map[string]*api.Node{}, cluster: cluster}
	}
	return s.nodes[cluster]
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
