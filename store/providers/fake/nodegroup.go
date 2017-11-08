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

type NodeGroupFileStore struct {
	container map[string]*api.NodeGroup
	cluster   string

	mux sync.Mutex
}

var _ store.NodeGroupStore = &NodeGroupFileStore{}

func (s *NodeGroupFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "nodeGroups")
}

func (s *NodeGroupFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *NodeGroupFileStore) List(opts metav1.ListOptions) ([]*api.NodeGroup, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*api.NodeGroup, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *NodeGroupFileStore) Get(name string) (*api.NodeGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, errors.New("missing node group name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	item, itemOK := s.container[s.resourceID(name)]
	if !itemOK {
		return nil, fmt.Errorf("NodeGroup `%s` does not exist", name)
	}
	return item, nil
}

func (s *NodeGroupFileStore) Create(obj *api.NodeGroup) (*api.NodeGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing node group")
	} else if obj.Name == "" {
		return nil, errors.New("missing node group name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; ok {
		return nil, fmt.Errorf("NodeGroup `%s` already exists", obj.Name)
	}

	s.container[id] = obj
	return obj, err
}

func (s *NodeGroupFileStore) Update(obj *api.NodeGroup) (*api.NodeGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing node group")
	} else if obj.Name == "" {
		return nil, errors.New("missing node group name")
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

func (s *NodeGroupFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing node group name")
	}
	delete(s.container, s.resourceID(name))
	return nil
}

func (s *NodeGroupFileStore) UpdateStatus(obj *api.NodeGroup) (*api.NodeGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing node group")
	} else if obj.Name == "" {
		return nil, errors.New("missing node group name")
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
		return nil, fmt.Errorf("NodeGroup `%s` does not exist", obj.Name)
	}
	existing.Status = obj.Status
	s.container[id] = existing
	return existing, err
}
