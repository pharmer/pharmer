package fake

import (
	"path/filepath"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type credentialFileStore struct {
	container map[string]*api.Credential

	mux sync.Mutex
}

var _ store.CredentialStore = &credentialFileStore{}

func (s *credentialFileStore) resourceHome() string {
	return "credentials"
}

func (s *credentialFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *credentialFileStore) List(opts metav1.ListOptions) ([]*api.Credential, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*api.Credential, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *credentialFileStore) Get(name string) (*api.Credential, error) {
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

func (s *credentialFileStore) Create(obj *api.Credential) (*api.Credential, error) {
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

func (s *credentialFileStore) Update(obj *api.Credential) (*api.Credential, error) {
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

	delete(s.container, s.resourceID(name))
	return nil
}
