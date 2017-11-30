package fake

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/pharmer/pharmer/store"
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
		return nil, nil, fmt.Errorf("SSH `id_%s.pub` does not exist", name)
	}

	privKey, privOK := s.container[s.pubKeyID(name)]
	if !privOK {
		return nil, nil, fmt.Errorf("SSH key `id_%s` does not exist", name)
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

	delete(s.container, s.pubKeyID(name))
	delete(s.container, s.privKeyID(name))
	return nil
}
