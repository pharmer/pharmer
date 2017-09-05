package vfs

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/appscode/pharmer/store"
	"github.com/graymeta/stow"
	"k8s.io/client-go/util/cert"
)

type CertificateFileStore struct {
	container stow.Container
	prefix    string
	cluster   string
}

var _ store.CertificateStore = &CertificateFileStore{}

func (s *CertificateFileStore) resourceHome() string {
	return filepath.Join(s.prefix, "clusters", s.cluster, "pki")
}

func (s *CertificateFileStore) certID(name string) string {
	return filepath.Join(s.resourceHome(), name+".crt")
}

func (s *CertificateFileStore) keyID(name string) string {
	return filepath.Join(s.resourceHome(), name+".key")
}

func (s *CertificateFileStore) Get(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("Missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("Missing certificate name")
	}

	itemCrt, err := s.container.Item(s.certID(name))
	if err != nil {
		return nil, nil, fmt.Errorf("Certificate `%s.crt` does not exist. Reason: %v", name, err)
	}
	r, err := itemCrt.Open()
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	crtPEM, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	crt, err := cert.ParseCertsPEM(crtPEM)
	if err != nil {
		return nil, nil, err
	}

	itemKey, err := s.container.Item(s.keyID(name))
	if err != nil {
		return nil, nil, fmt.Errorf("Certificate key `%s.key` does not exist. Reason: %v", name, err)
	}
	r2, err := itemKey.Open()
	if err != nil {
		return nil, nil, err
	}
	defer r2.Close()
	keyPEM, err := ioutil.ReadAll(r2)
	if err != nil {
		return nil, nil, err
	}
	key, err := cert.ParsePrivateKeyPEM(keyPEM)
	if err != nil {
		return nil, nil, err
	}
	return crt[0], key.(*rsa.PrivateKey), nil
}

func (s *CertificateFileStore) Create(name string, crt *x509.Certificate, key *rsa.PrivateKey) error {
	if s.cluster == "" {
		return errors.New("Missing cluster name")
	}
	if crt == nil {
		return errors.New("Missing certificate")
	} else if key == nil {
		return errors.New("Missing certificate key")
	}

	id := s.certID(name)
	_, err := s.container.Item(id)
	if err == nil {
		return fmt.Errorf("Certificate `%s.crt` already exists. Reason: %v.", name, err)
	}
	bufCert := bytes.NewBuffer(cert.EncodeCertPEM(crt))
	_, err = s.container.Put(id, bufCert, int64(bufCert.Len()), nil)
	if err != nil {
		return fmt.Errorf("Failed to store certificate `%s.crt`. Reason: %v.", name, err)
	}

	id = s.keyID(name)
	_, err = s.container.Item(id)
	if err == nil {
		return fmt.Errorf("Certificate `%s.key` already exists. Reason: %v.", name, err)
	}
	bufKey := bytes.NewBuffer(cert.EncodePrivateKeyPEM(key))
	_, err = s.container.Put(id, bufKey, int64(bufKey.Len()), nil)
	if err != nil {
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
