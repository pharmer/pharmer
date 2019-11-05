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
	"path/filepath"
	"sync"

	"pharmer.dev/pharmer/store"

	"github.com/pkg/errors"
)

type sshKeyFileStore struct {
	container map[string][]byte
	cluster   string

	mux sync.Mutex
}

var _ store.SSHKeyStore = &sshKeyFileStore{}

func (s *sshKeyFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "ssh")
}

func (s *sshKeyFileStore) pubKeyID(name string) string {
	return filepath.Join(s.resourceHome(), "id_"+name+".pub")
}

func (s *sshKeyFileStore) privKeyID(name string) string {
	return filepath.Join(s.resourceHome(), "id_"+name)
}

func (s *sshKeyFileStore) Get(name string) ([]byte, []byte, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("missing ssh key name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	pubKey, pubOK := s.container[s.pubKeyID(name)]
	if !pubOK {
		return nil, nil, errors.Errorf("SSH `id_%s.pub` does not exist", name)
	}

	privKey, privOK := s.container[s.privKeyID(name)]
	if !privOK {
		return nil, nil, errors.Errorf("SSH key `id_%s` does not exist", name)
	}
	return pubKey, privKey, nil
}

func (s *sshKeyFileStore) Create(name string, pubKey, privKey []byte) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if len(pubKey) == 0 {
		return errors.New("empty ssh public key")
	} else if len(privKey) == 0 {
		return errors.New("empty ssh private key")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	s.container[s.pubKeyID(name)] = pubKey
	s.container[s.privKeyID(name)] = privKey
	return nil
}

func (s *sshKeyFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing ssh key name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	_, exist := s.container[s.pubKeyID(name)]
	if !exist {
		return errors.New("sshkey not found")
	}

	delete(s.container, s.pubKeyID(name))
	delete(s.container, s.privKeyID(name))
	return nil
}
