package fake

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/store"
)

const (
	UID = "fake"
)

func init() {
	store.RegisterProvider(UID, func(ctx context.Context, cfg *api.PharmerConfig) (store.Interface, error) {
		return New(), nil
	})
}

type FakeStore struct {
	credentials  store.CredentialStore
	clusters     store.ClusterStore
	nodeGroups   map[string]store.NodeGroupStore
	certificates map[string]store.CertificateStore
	sshKeys      map[string]store.SSHKeyStore

	owner string

	mux sync.Mutex
}

var _ store.Interface = &FakeStore{}

func New() store.Interface {
	return &FakeStore{
		nodeGroups:   map[string]store.NodeGroupStore{},
		certificates: map[string]store.CertificateStore{},
		sshKeys:      map[string]store.SSHKeyStore{},
	}
}

func (s *FakeStore) Owner(id string) store.ResourceInterface {
	ret := *s
	ret.owner = id
	return &ret
}

func (s *FakeStore) Credentials() store.CredentialStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.credentials == nil {
		s.credentials = &credentialFileStore{container: map[string]*api.Credential{}}
	}

	return s.credentials
}

func (s *FakeStore) Clusters() store.ClusterStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.clusters == nil {
		s.clusters = &clusterFileStore{container: map[string]*api.Cluster{}}
	}
	return s.clusters
}

func (s *FakeStore) NodeGroups(cluster string) store.NodeGroupStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.nodeGroups[cluster]; !found {
		s.nodeGroups[cluster] = &nodeGroupFileStore{container: map[string]*api.NodeGroup{}, cluster: cluster}
	}
	return s.nodeGroups[cluster]
}

func (s *FakeStore) Certificates(cluster string) store.CertificateStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.certificates[cluster]; !found {
		s.certificates[cluster] = &certificateFileStore{certs: map[string]*x509.Certificate{}, keys: map[string]*rsa.PrivateKey{}, cluster: cluster}
	}
	return s.certificates[cluster]
}

func (s *FakeStore) SSHKeys(cluster string) store.SSHKeyStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, found := s.sshKeys[cluster]; !found {
		s.sshKeys[cluster] = &sshKeyFileStore{container: map[string][]byte{}, cluster: cluster}
	}
	return s.sshKeys[cluster]
}
