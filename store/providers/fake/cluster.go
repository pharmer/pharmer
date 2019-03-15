package fake

import (
	"path/filepath"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterFileStore struct {
	container map[string]*api.Cluster

	mux sync.Mutex
}

var _ store.ClusterStore = &clusterFileStore{}

func (s *clusterFileStore) resourceHome() string {
	return "clusters"
}

func (s *clusterFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *clusterFileStore) List(opts metav1.ListOptions) ([]*api.Cluster, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*api.Cluster, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *clusterFileStore) Get(name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("missing cluster name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	existing, ok := s.container[s.resourceID(name)]
	if !ok {
		return nil, errors.Errorf("cluster `%s` does not exist", name)
	}
	return existing, nil
}

func (s *clusterFileStore) Create(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; ok {
		return nil, errors.Errorf("cluster `%s` already exists", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *clusterFileStore) Update(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	id := s.resourceID(obj.Name)
	if _, ok := s.container[id]; !ok {
		return nil, errors.Errorf("cluster `%s` does not exist", obj.Name)
	}
	s.container[id] = obj
	return obj, err
}

func (s *clusterFileStore) Delete(name string) error {
	if name == "" {
		return errors.New("missing cluster name")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.container, s.resourceID(name))
	return nil
}

func (s *clusterFileStore) UpdateStatus(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("missing cluster name")
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
		return nil, errors.Errorf("cluster `%s` does not exist", obj.Name)
	}
	existing.Status = obj.Status
	s.container[id] = existing
	return existing, err
}
