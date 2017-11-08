package fake

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/appscode/pharmer/store"
)

type SSHKeyFileStore struct {
	container map[string][]byte
	cluster   string

	mux sync.Mutex
}

var _ store.SSHKeyStore = &SSHKeyFileStore{}

func (s *SSHKeyFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "ssh")
}

func (s *SSHKeyFileStore) pubKeyID(name string) string {
	return filepath.Join(s.resourceHome(), "id_"+name+".pub")
}

func (s *SSHKeyFileStore) privKeyID(name string) string {
	return filepath.Join(s.resourceHome(), "id_"+name)
}

func (s *SSHKeyFileStore) Get(name string) ([]byte, []byte, error) {
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
		return nil, nil, fmt.Errorf("SSH `id_%s.pub` does not exist", name)
	}

	privKey, privOK := s.container[s.pubKeyID(name)]
	if !privOK {
		return nil, nil, fmt.Errorf("SSH key `id_%s` does not exist", name)
	}
	return pubKey, privKey, nil
}

func (s *SSHKeyFileStore) Create(name string, pubKey, privKey []byte) error {
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

func (s *SSHKeyFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing ssh key name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.container, s.pubKeyID(name))
	delete(s.container, s.privKeyID(name))
	return nil
}
