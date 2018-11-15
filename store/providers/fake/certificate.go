package fake

import (
	"crypto/rsa"
	"crypto/x509"
	"path/filepath"
	"sync"

	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
)

type certificateFileStore struct {
	certs   map[string]*x509.Certificate
	keys    map[string]*rsa.PrivateKey
	cluster string
	owner   string

	mux sync.Mutex
}

var _ store.CertificateStore = &certificateFileStore{}

func (s *certificateFileStore) With(owner string) store.CertificateStore {
	s.owner = owner
	return s
}

func (s *certificateFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "pki")
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

	s.mux.Lock()
	defer s.mux.Unlock()

	crt, certOK := s.certs[s.certID(name)]
	if !certOK {
		return nil, nil, errors.Errorf("certificate `%s.crt` does not exist", name)
	}

	key, keyOK := s.keys[s.keyID(name)]
	if !keyOK {
		return nil, nil, errors.Errorf("certificate key `%s.key` does not exist", name)
	}
	return crt, key, nil
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

	s.mux.Lock()
	defer s.mux.Unlock()

	s.certs[s.certID(name)] = crt
	s.keys[s.keyID(name)] = key
	return nil
}

func (s *certificateFileStore) Delete(name string) error {
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
