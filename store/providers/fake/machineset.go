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

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type machineSetFileStore struct {
	container map[string]*clusterapi.MachineSet
	cluster   string

	mux sync.Mutex
}

var _ store.MachineSetStore = &machineSetFileStore{}

func (s *machineSetFileStore) resourceHome() string {
	return filepath.Join("clusters", s.cluster, "machineset")
}

func (s *machineSetFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *machineSetFileStore) List(opts metav1.ListOptions) ([]*clusterapi.MachineSet, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	result := make([]*clusterapi.MachineSet, 0)
	for k := range s.container {
		result = append(result, s.container[k])
	}
	return result, nil
}

func (s *machineSetFileStore) Get(name string) (*clusterapi.MachineSet, error) {
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

func (s *machineSetFileStore) Create(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error) {
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

func (s *machineSetFileStore) Update(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error) {
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

	_, exist := s.container[s.resourceID(name)]
	if !exist {
		return errors.New("machineset item not found")
	}

	delete(s.container, s.resourceID(name))
	return nil
}

func (s *machineSetFileStore) UpdateStatus(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error) {
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
