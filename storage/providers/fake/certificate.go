package fake

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/appscode/pharmer/storage"
)

type CertificateFileStore struct {
	container map[string][]byte
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

	certPEM, certOK := s.container[s.certID(name)]
	if !certOK {
		return nil, nil, fmt.Errorf("Certificate `%s.crt` does not exist.", name)
	}

	keyPEM, keyOK := s.container[s.keyID(name)]
	if !keyOK {
		return nil, nil, fmt.Errorf("Certificate key `%s.key` does not exist.", name)
	}
	return certPEM, keyPEM, nil
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

	s.container[s.certID(name)] = certPEM
	s.container[s.keyID(name)] = keyPEM
	return nil
}

func (s *CertificateFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("Missing cluster name")
	}
	if name == "" {
		return errors.New("Missing certificate name")
	}

	delete(s.container, s.certID(name))
	delete(s.container, s.keyID(name))
	return nil
}
