package fake

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/store"
)

type InstanceFileStore struct {
	container map[string]*api.Instance
	cluster   string

	mux sync.Mutex
}

var _ store.InstanceStore = &InstanceFileStore{}

func (s *InstanceFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "instances")
}

func (s *InstanceFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *InstanceFileStore) List(opts api.ListOptions) ([]*api.Instance, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*api.Instance, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *InstanceFileStore) Get(name string) (*api.Instance, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if name == "" {
		return nil, errors.New("Missing instance name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	item, itemOK := s.container[s.resourceID(name)]
	if !itemOK {
		return nil, fmt.Errorf("Instance `%s` does not exist.", name)
	}
	return item, nil
}

func (s *InstanceFileStore) Create(obj *api.Instance) (*api.Instance, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; ok {
		return nil, fmt.Errorf("Instance `%s` already exists", obj.Name)
	}

	s.container[id] = obj
	return obj, err
}

func (s *InstanceFileStore) Update(obj *api.Instance) (*api.Instance, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	s.container[id] = obj
	return obj, err
}

func (s *InstanceFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("Missing cluster name")
	}
	if name == "" {
		return errors.New("Missing instance name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.container, s.resourceID(name))
	return nil
}

func (s *InstanceFileStore) UpdateStatus(obj *api.Instance) (*api.Instance, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	existing, itemOK := s.container[id]
	if !itemOK {
		return nil, fmt.Errorf("Instance `%s` does not exist.", obj.Name)
	}
	existing.Status = obj.Status
	s.container[id] = existing
	return existing, err
}
