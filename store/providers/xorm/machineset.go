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
package xorm

import (
	"pharmer.dev/pharmer/store"

	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type machineSetXormStore struct {
	engine  *xorm.Engine
	cluster string
}

var _ store.MachineSetStore = &machineSetXormStore{}

func (s *machineSetXormStore) List(opts metav1.ListOptions) ([]*clusterapi.MachineSet, error) {
	return nil, nil
}

func (s *machineSetXormStore) Get(name string) (*clusterapi.MachineSet, error) {
	return nil, nil
}

func (s *machineSetXormStore) Create(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error) {
	return nil, nil
	/*if s.cluster == "" {
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

	found, err := s.engine.Get(&NodeGroup{Name: obj.Name, ClusterName: s.cluster})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if found {
		return nil, errors.Errorf("node group `%s` already exists", obj.Name)
	}

	obj.CreationTimestamp = metav1.Time{Time: time.Now()}
	nodeGroup, err := encodeNodeGroup(obj)
	if err != nil {
		return nil, err
	}

	_, err = s.engine.Insert(nodeGroup)
	return obj, err*/
}

func (s *machineSetXormStore) Update(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error) {
	return nil, nil
}

func (s *machineSetXormStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing node group name")
	}
	_, err := s.engine.Delete(&machineSetXormStore{})
	return err
}

func (s *machineSetXormStore) UpdateStatus(obj *clusterapi.MachineSet) (*clusterapi.MachineSet, error) {
	return nil, nil
}
