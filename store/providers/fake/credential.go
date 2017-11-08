package fake

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CredentialFileStore struct {
	container map[string]*api.Credential

	mux sync.Mutex
}

var _ store.CredentialStore = &CredentialFileStore{}

func (s *CredentialFileStore) resourceHome() string {
	return "credentials"
}

func (s *CredentialFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *CredentialFileStore) List(opts metav1.ListOptions) ([]*api.Credential, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*api.Credential, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *CredentialFileStore) Get(name string) (*api.Credential, error) {
	if name == "" {
		return nil, errors.New("missing credential name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	existing, ok := s.container[s.resourceID(name)]
	if !ok {
		return nil, fmt.Errorf("credential `%s` does not exist", name)
	}
	return existing, nil
}

func (s *CredentialFileStore) Create(obj *api.Credential) (*api.Credential, error) {
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
		return nil, fmt.Errorf("credential `%s` already exists", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *CredentialFileStore) Update(obj *api.Credential) (*api.Credential, error) {
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
		return nil, fmt.Errorf("credential `%s` does not exist", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *CredentialFileStore) Delete(name string) error {
	if name == "" {
		return errors.New("missing credential name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.container, s.resourceID(name))
	return nil
}
