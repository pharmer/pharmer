package fake

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/store"
)

type InstanceGroupFileStore struct {
	container map[string]*api.InstanceGroup
	cluster   string

	mux sync.Mutex
}

var _ store.InstanceGroupStore = &InstanceGroupFileStore{}

func (s *InstanceGroupFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "instancegroups")
}

func (s *InstanceGroupFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *InstanceGroupFileStore) List(opts api.ListOptions) ([]*api.InstanceGroup, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*api.InstanceGroup, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *InstanceGroupFileStore) Get(name string) (*api.InstanceGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if name == "" {
		return nil, errors.New("Missing instance group name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	item, itemOK := s.container[s.resourceID(name)]
	if !itemOK {
		return nil, fmt.Errorf("InstanceGroup `%s` does not exist.", name)
	}
	return item, nil
}

func (s *InstanceGroupFileStore) Create(obj *api.InstanceGroup) (*api.InstanceGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance group")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance group name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; ok {
		return nil, fmt.Errorf("InstanceGroup `%s` already exists", obj.Name)
	}

	s.container[id] = obj
	return obj, err
}

func (s *InstanceGroupFileStore) Update(obj *api.InstanceGroup) (*api.InstanceGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance group")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance group name")
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

func (s *InstanceGroupFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("Missing cluster name")
	}
	if name == "" {
		return errors.New("Missing instance group name")
	}
	delete(s.container, s.resourceID(name))
	return nil
}

func (s *InstanceGroupFileStore) UpdateStatus(obj *api.InstanceGroup) (*api.InstanceGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance group")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance group name")
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
		return nil, fmt.Errorf("InstanceGroup `%s` does not exist.", obj.Name)
	}
	existing.Status = obj.Status
	s.container[id] = existing
	return existing, err
}
