package fake

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/storage"
)

type CredentialFileStore struct {
	container map[string]*api.Credential
}

var _ storage.CredentialStore = &CredentialFileStore{}

func (s *CredentialFileStore) resourceHome() string {
	return "credentials"
}

func (s *CredentialFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *CredentialFileStore) List(opts api.ListOptions) ([]*api.Credential, error) {
	result := make([]*api.Credential, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *CredentialFileStore) Get(name string) (*api.Credential, error) {
	if name == "" {
		return nil, errors.New("Missing credential name")
	}

	existing, ok := s.container[s.resourceID(name)]
	if !ok {
		return nil, fmt.Errorf("Credential `%s` does not exist.", name)
	}
	return existing, nil
}

func (s *CredentialFileStore) Create(obj *api.Credential) (*api.Credential, error) {
	if obj == nil {
		return nil, errors.New("Missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("Missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; ok {
		return nil, fmt.Errorf("Credential `%s` already exists", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *CredentialFileStore) Update(obj *api.Credential) (*api.Credential, error) {
	if obj == nil {
		return nil, errors.New("Missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("Missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; !ok {
		return nil, fmt.Errorf("Credential `%s` does not exist.", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *CredentialFileStore) Delete(name string) error {
	if name == "" {
		return errors.New("Missing credential name")
	}
	delete(s.container, s.resourceID(name))
	return nil
}
