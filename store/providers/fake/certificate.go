package fake

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/appscode/pharmer/store"
)

type CertificateFileStore struct {
	certs   map[string]*x509.Certificate
	keys    map[string]*rsa.PrivateKey
	cluster string

	mux sync.Mutex
}

var _ store.CertificateStore = &CertificateFileStore{}

func (s *CertificateFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "pki")
}

func (s *CertificateFileStore) certID(name string) string {
	return filepath.Join(s.resourceHome(), name+".crt")
}

func (s *CertificateFileStore) keyID(name string) string {
	return filepath.Join(s.resourceHome(), name+".key")
}

func (s *CertificateFileStore) Get(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("missing certificate name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	crt, certOK := s.certs[s.certID(name)]
	if !certOK {
		return nil, nil, fmt.Errorf("Certificate `%s.crt` does not exist.", name)
	}

	key, keyOK := s.keys[s.keyID(name)]
	if !keyOK {
		return nil, nil, fmt.Errorf("Certificate key `%s.key` does not exist.", name)
	}
	return crt, key, nil
}

func (s *CertificateFileStore) Create(name string, crt *x509.Certificate, key *rsa.PrivateKey) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if crt == nil {
		return errors.New("missing certificate")
	} else if key == nil {
		return errors.New("missing certificate key")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	s.certs[s.certID(name)] = crt
	s.keys[s.keyID(name)] = key
	return nil
}

func (s *CertificateFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing certificate name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.certs, s.certID(name))
	delete(s.keys, s.keyID(name))
	return nil
}
