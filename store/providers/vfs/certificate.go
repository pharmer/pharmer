package vfs

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
	"path/filepath"

	"github.com/graymeta/stow"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/cert"
)

type certificateFileStore struct {
	container stow.Container
	prefix    string
	cluster   string
	owner     string
}

var _ store.CertificateStore = &certificateFileStore{}

func (s *certificateFileStore) resourceHome() string {
	return filepath.Join(s.owner, s.prefix, "clusters", s.cluster, "pki")
}

func (s *certificateFileStore) certID(name string) string {
	return filepath.Join(s.resourceHome(), name+".crt")
}

func (s *certificateFileStore) keyID(name string) string {
	return filepath.Join(s.resourceHome(), name+".key")
}

func (s *certificateFileStore) Get(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("missing certificate name")
	}

	itemCrt, err := s.container.Item(s.certID(name))
	if err != nil {
		return nil, nil, errors.Errorf("certificate `%s.crt` does not exist. Reason: %v", name, err)
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
		return nil, nil, errors.Errorf("certificate key `%s.key` does not exist. Reason: %v", name, err)
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

func (s *certificateFileStore) Create(name string, crt *x509.Certificate, key *rsa.PrivateKey) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if crt == nil {
		return errors.New("missing certificate")
	} else if key == nil {
		return errors.New("missing certificate key")
	}

	id := s.certID(name)
	_, err := s.container.Item(id)
	if err == nil {
		return errors.Errorf("certificate `%s.crt` already exists. Reason: %v", name, err)
	}
	bufCert := bytes.NewBuffer(cert.EncodeCertPEM(crt))
	_, err = s.container.Put(id, bufCert, int64(bufCert.Len()), nil)
	if err != nil {
		return errors.Errorf("failed to store certificate `%s.crt`. Reason: %v", name, err)
	}

	id = s.keyID(name)
	_, err = s.container.Item(id)
	if err == nil {
		return errors.Errorf("certificate `%s.key` already exists. Reason: %v", name, err)
	}
	bufKey := bytes.NewBuffer(cert.EncodePrivateKeyPEM(key))
	_, err = s.container.Put(id, bufKey, int64(bufKey.Len()), nil)
	if err != nil {
		return errors.Errorf("failed to store certificate key `%s.key`. Reason: %v", name, err)
	}

	return nil
}

func (s *certificateFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing certificate name")
	}

	certID := s.certID(name)
	certItem, err := s.container.Item(certID)
	if err != nil {
		return errors.Errorf("failed to get item %s. Reason: %v", name, err)
	}

	if err := s.container.RemoveItem(certItem.ID()); err != nil {
		return errors.Errorf("failed to delete certificate %s.crt. Reason: %v", name, err)
	}

	keyID := s.keyID(name)
	keyItem, err := s.container.Item(keyID)
	if err != nil {
		return errors.Errorf("failed to get item %s. Reason: %v", name, err)
	}

	if err := s.container.RemoveItem(keyItem.ID()); err != nil {
		return errors.Errorf("failed to delete certificate key %s.key. Reason: %v", name, err)
	}

	return nil
}
