package xorm

import (
	"github.com/go-xorm/xorm"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type machineSetXormStore struct {
	engine  *xorm.Engine
	cluster string
}

var _ store.MachineSetStore = &machineSetXormStore{}

func (s *machineSetXormStore) List(opts metav1.ListOptions) ([]*clusterv1.MachineSet, error) {
	return nil, nil
}

func (s *machineSetXormStore) Get(name string) (*clusterv1.MachineSet, error) {
	return nil, nil
}

func (s *machineSetXormStore) Create(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error) {
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

func (s *machineSetXormStore) Update(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error) {
	return nil, nil
}

func (s *machineSetXormStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing node group name")
	}
	_, err := s.engine.Delete(&NodeGroup{Name: name, ClusterName: s.cluster})
	return err
}

func (s *machineSetXormStore) UpdateStatus(obj *clusterv1.MachineSet) (*clusterv1.MachineSet, error) {
	return nil, nil
}
