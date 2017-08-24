package vfs

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/appscode/pharmer/storage"
	"github.com/graymeta/stow"
)

type CertificateFileStore struct {
	container stow.Container
	cluster   string
}

var _ storage.CertificateStore = &CertificateFileStore{}

func (s *CertificateFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "pki")
}

func (s *CertificateFileStore) certID(name string) string {
	return filepath.Join(s.resourceHome(), name+".crt")
}

func (s *CertificateFileStore) keyID(name string) string {
	return filepath.Join(s.resourceHome(), name+".key")
}

func (s *CertificateFileStore) Get(name string) ([]byte, []byte, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("Missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("Missing certificate name")
	}

	crt, err := s.container.Item(s.certID(name))
	if err != nil {
		return nil, nil, fmt.Errorf("Certificate `%s.crt` does not exist. Reason: %v", name, err)
	}
	r, err := crt.Open()
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	crtPEM, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	key, err := s.container.Item(s.certID(name))
	if err != nil {
		return nil, nil, fmt.Errorf("Certificate key `%s.key` does not exist. Reason: %v", name, err)
	}
	r2, err := key.Open()
	if err != nil {
		return nil, nil, err
	}
	defer r2.Close()
	keyPEM, err := ioutil.ReadAll(r2)
	if err != nil {
		return nil, nil, err
	}
	return crtPEM, keyPEM, nil
}

func (s *CertificateFileStore) Create(name string, certPEM, keyPEM []byte) error {
	if s.cluster == "" {
		return errors.New("Missing cluster name")
	}
	if len(certPEM) == 0 {
		return errors.New("Empty certificate")
	} else if len(keyPEM) == 0 {
		return errors.New("Empty certificate key")
	}

	id := s.certID(name)
	_, err := s.container.Item(id)
	if err == nil {
		return fmt.Errorf("Certificate `%s.crt` already exists. Reason: %v.", name, err)
	}
	_, err = s.container.Put(id, bytes.NewBuffer(certPEM), int64(len(certPEM)), nil)
	if err == nil {
		return fmt.Errorf("Failed to store certificate `%s.crt`. Reason: %v.", name, err)
	}

	id = s.keyID(name)
	_, err = s.container.Item(id)
	if err == nil {
		return fmt.Errorf("Certificate `%s.key` already exists. Reason: %v.", name, err)
	}
	_, err = s.container.Put(id, bytes.NewBuffer(keyPEM), int64(len(keyPEM)), nil)
	if err == nil {
		return fmt.Errorf("Failed to store certificate key `%s.key`. Reason: %v.", name, err)
	}

	return nil
}

func (s *CertificateFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("Missing cluster name")
	}
	if name == "" {
		return errors.New("Missing certificate name")
	}

	err := s.container.RemoveItem(s.certID(name))
	if err != nil {
		return fmt.Errorf("Failed to delete certificate %s.crt. Reason: %v", name, err)
	}
	err = s.container.RemoveItem(s.keyID(name))
	if err != nil {
		return fmt.Errorf("Failed to delete certificate key %s.key. Reason: %v", name, err)
	}
	return nil
}
