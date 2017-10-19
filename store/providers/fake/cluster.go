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

type ClusterFileStore struct {
	container map[string]*api.Cluster

	mux sync.Mutex
}

var _ store.ClusterStore = &ClusterFileStore{}

func (s *ClusterFileStore) resourceHome() string {
	return "clusters"
}

func (s *ClusterFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *ClusterFileStore) List(opts metav1.ListOptions) ([]*api.Cluster, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*api.Cluster, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *ClusterFileStore) Get(name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("Missing cluster name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	existing, ok := s.container[s.resourceID(name)]
	if !ok {
		return nil, fmt.Errorf("Cluster `%s` does not exist.", name)
	}
	return existing, nil
}

func (s *ClusterFileStore) Create(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("Missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("Missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; ok {
		return nil, fmt.Errorf("Cluster `%s` already exists", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *ClusterFileStore) Update(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("Missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("Missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; !ok {
		return nil, fmt.Errorf("Cluster `%s` does not exist.", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *ClusterFileStore) Delete(name string) error {
	if name == "" {
		return errors.New("Missing cluster name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.container, s.resourceID(name))
	return nil
}

func (s *ClusterFileStore) UpdateStatus(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("Missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("Missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	existing, ok := s.container[id]
	if !ok {
		return nil, fmt.Errorf("Cluster `%s` does not exist.", obj.Name)
	}
	existing.Status = obj.Status
	s.container[id] = existing
	return existing, err
}
