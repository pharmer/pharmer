package fake

import (
	"crypto/rsa"
	"crypto/x509"
	"sync"

	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	UID = "fake"
)

func init() {
	store.RegisterProvider(UID, func(cfg *api.PharmerConfig) (store.Interface, error) {
		return New(), nil
	})
}

type FakeStore struct {
	clusters map[string]*api.Cluster
	//credentials  map[string]store.CredentialStore
	credentials  store.CredentialStore
	machineSet   map[string]store.MachineSetStore
	machine      map[string]store.MachineStore
	certificates map[string]store.CertificateStore
	sshKeys      map[string]store.SSHKeyStore

	operations store.OperationStore

	mux sync.Mutex
}

var _ store.Interface = &FakeStore{}

func New() store.Interface {
	return &FakeStore{
		clusters:     map[string]*api.Cluster{},
		machineSet:   map[string]store.MachineSetStore{},
		machine:      map[string]store.MachineStore{},
		certificates: map[string]store.CertificateStore{},
		sshKeys:      map[string]store.SSHKeyStore{},
		//operations:   store.OperationStore{},
	}
}

func (s *FakeStore) Owner(id int64) store.ResourceInterface {
	return s
}

func (s *FakeStore) Credentials() store.CredentialStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.credentials == nil {
		s.credentials = &credentialFileStore{container: map[string]*cloudapi.Credential{}}
	}

	return s.credentials
}

func (s *FakeStore) Clusters() store.ClusterStore {
	return &clusterFileStore{container: s.clusters}
}

func (s *FakeStore) MachineSet(cluster string) store.MachineSetStore {
	s.mux.Lock()
	defer s.mux.Unlock()
	if _, found := s.machineSet[cluster]; !found {
		s.machineSet[cluster] = &machineSetFileStore{container: map[string]*clusterv1.MachineSet{}, cluster: cluster}
	}
	return s.machineSet[cluster]
}

func (s *FakeStore) Machine(cluster string) store.MachineStore {
	s.mux.Lock()
	defer s.mux.Unlock()
	if _, found := s.machine[cluster]; !found {
		s.machine[cluster] = &machineFileStore{container: map[string]*clusterv1.Machine{}, cluster: cluster}
	}
	return s.machine[cluster]
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

func (s *FakeStore) Operations() store.OperationStore {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.operations = &operationFileStore{container: map[string][]byte{}}

	return s.operations
}
