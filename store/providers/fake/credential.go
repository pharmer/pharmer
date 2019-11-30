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
	"path/filepath"
	"sync"

	cloudapi "pharmer.dev/cloud/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type credentialFileStore struct {
	container map[string]*cloudapi.Credential

	mux sync.Mutex
}

var _ store.CredentialStore = &credentialFileStore{}

func (s *credentialFileStore) resourceHome() string {
	return "credentials"
}

func (s *credentialFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *credentialFileStore) List(opts metav1.ListOptions) ([]*cloudapi.Credential, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*cloudapi.Credential, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *credentialFileStore) Get(name string) (*cloudapi.Credential, error) {
	if name == "" {
		return nil, errors.New("missing credential name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	existing, ok := s.container[s.resourceID(name)]
	if !ok {
		return nil, errors.Errorf("credential `%s` does not exist", name)
	}
	return existing, nil
}

func (s *credentialFileStore) Create(obj *cloudapi.Credential) (*cloudapi.Credential, error) {
	if obj == nil {
		return nil, errors.New("missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; ok {
		return nil, errors.Errorf("credential `%s` already exists", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *credentialFileStore) Update(obj *cloudapi.Credential) (*cloudapi.Credential, error) {
	if obj == nil {
		return nil, errors.New("missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; !ok {
		return nil, errors.Errorf("credential `%s` does not exist", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *credentialFileStore) Delete(name string) error {
	if name == "" {
		return errors.New("missing credential name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	_, exist := s.container[s.resourceID(name)]
	if !exist {
		return errors.New("credential not found")
	}

	delete(s.container, s.resourceID(name))
	return nil
}
