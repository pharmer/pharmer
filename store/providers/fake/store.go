/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package fake

import (
	"crypto/rsa"
	"crypto/x509"
	"sync"

	cloudapi "pharmer.dev/cloud/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"

	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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
		s.machineSet[cluster] = &machineSetFileStore{container: map[string]*clusterapi.MachineSet{}, cluster: cluster}
	}
	return s.machineSet[cluster]
}

func (s *FakeStore) Machine(cluster string) store.MachineStore {
	s.mux.Lock()
	defer s.mux.Unlock()
	if _, found := s.machine[cluster]; !found {
		s.machine[cluster] = &machineFileStore{container: map[string]*clusterapi.Machine{}, cluster: cluster}
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
