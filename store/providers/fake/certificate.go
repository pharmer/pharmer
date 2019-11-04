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
	"crypto/rsa"
	"crypto/x509"
	"path/filepath"
	"sync"

	"pharmer.dev/pharmer/store"

	"github.com/pkg/errors"
)

type certificateFileStore struct {
	certs   map[string]*x509.Certificate
	keys    map[string]*rsa.PrivateKey
	cluster string

	mux sync.Mutex
}

var _ store.CertificateStore = &certificateFileStore{}

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

	_, exist := s.certs[s.certID(name)]
	if !exist {
		return errors.New("certificate not found")
	}

	delete(s.certs, s.certID(name))
	delete(s.keys, s.keyID(name))
	return nil
}
