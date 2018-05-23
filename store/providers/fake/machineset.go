package fake

import (
	"path/filepath"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type machineSetFileStore struct {
	container map[string]*clusterv1.MachineSet
	cluster   string

	mux sync.Mutex
}

var _ store.MachineSetStore = &machineSetFileStore{}

func (s *machineSetFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "nodeGroups")
}

func (s *machineSetFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *machineSetFileStore) List(opts metav1.ListOptions) ([]*clusterv1.MachineSet, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*clusterv1.MachineSet, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *machineSetFileStore) Get(name string) (*clusterv1.MachineSet, error) {
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
		return nil, errors.Errorf("NodeGroup `%s` does not exist", name)
	}
	return item, nil
}

func (s *machineSetFileStore) Create(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error) {
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
		return nil, errors.Errorf("NodeGroup `%s` already exists", obj.Name)
	}

	s.container[id] = obj
	return obj, err
}

func (s *machineSetFileStore) Update(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error) {
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

func (s *machineSetFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing node group name")
	}
	delete(s.container, s.resourceID(name))
	return nil
}

func (s *machineSetFileStore) UpdateStatus(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error) {
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
		return nil, errors.Errorf("NodeGroup `%s` does not exist", obj.Name)
	}
	existing.Status = obj.Status
	s.container[id] = existing
	return existing, err
}
