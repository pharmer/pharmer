package vfs

import (
	"bytes"
	"io/ioutil"
	"path/filepath"

	"github.com/graymeta/stow"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
)

type sshKeyFileStore struct {
	container stow.Container
	prefix    string
	cluster   string
	owner     string
}

var _ store.SSHKeyStore = &sshKeyFileStore{}

func (s *sshKeyFileStore) resourceHome() string {
	return filepath.Join(s.owner, s.prefix, "clusters", s.cluster, "ssh")
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

	pub, err := s.container.Item(s.pubKeyID(name))
	if err != nil {
		return nil, nil, errors.Errorf("SSH `id_%s.pub` does not exist. Reason: %v", name, err)
	}
	r, err := pub.Open()
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	pubKey, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	priv, err := s.container.Item(s.privKeyID(name))
	if err != nil {
		return nil, nil, errors.Errorf("SSH key `id_%s` does not exist. Reason: %v", name, err)
	}
	r2, err := priv.Open()
	if err != nil {
		return nil, nil, err
	}
	defer r2.Close()
	privKey, err := ioutil.ReadAll(r2)
	if err != nil {
		return nil, nil, err
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

	id := s.pubKeyID(name)
	_, err := s.container.Item(id)
	if err == nil {
		return errors.Errorf("SSH `id_%s.pub` already exists. Reason: %v", name, err)
	}
	_, err = s.container.Put(id, bytes.NewBuffer(pubKey), int64(len(pubKey)), nil)
	if err != nil {
		return errors.Errorf("failed to store ssh public key `id_%s.pub`. Reason: %v", name, err)
	}

	id = s.privKeyID(name)
	_, err = s.container.Item(id)
	if err == nil {
		return errors.Errorf("SSH `id_%s` already exists. Reason: %v", name, err)
	}
	_, err = s.container.Put(id, bytes.NewBuffer(privKey), int64(len(privKey)), nil)
	if err != nil {
		return errors.Errorf("failed to store ssh private key `id_%s`. Reason: %v", name, err)
	}

	return nil
}

func (s *sshKeyFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing ssh key name")
	}

	pubKeyID := s.pubKeyID(name)
	pubKeyItem, err := s.container.Item(pubKeyID)
	if err != nil {
		return errors.Errorf("failed to get item %s. Reason: %v", name, err)
	}

	if err := s.container.RemoveItem(pubKeyItem.ID()); err != nil {
		return errors.Errorf("failed to delete ssh public key id_%s.pub. Reason: %v", name, err)
	}

	privKeyID := s.privKeyID(name)
	privKeyItem, err := s.container.Item(privKeyID)
	if err != nil {
		return errors.Errorf("failed to get item %s. Reason: %v", name, err)
	}

	err = s.container.RemoveItem(privKeyItem.ID())
	if err != nil {
		return errors.Errorf("failed to delete ssh private key id_%s. Reason: %v", name, err)
	}
	return nil
}
