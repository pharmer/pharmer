package xorm

import (
	"time"

	"github.com/go-xorm/xorm"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type machineXormStore struct {
	engine  *xorm.Engine
	cluster string
	owner   string
}

var _ store.MachineStore = &machineXormStore{}

func (s *machineXormStore) List(opts metav1.ListOptions) ([]*clusterv1.Machine, error) {
	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	result := make([]*clusterv1.Machine, 0)
	var machines []Machine
	err = s.engine.Where(`"cluster_id" = ?`, cluster.Id).Find(&machines)
	if err != nil {
		return nil, err
	}

	for _, m := range machines {
		decode, err := decodeMachine(&m)
		if err != nil {
			return nil, errors.Errorf("failed to list machines. Reason: %v", err)
		}
		result = append(result, decode)
	}
	return result, nil
}

func (s *machineXormStore) Get(name string) (*clusterv1.Machine, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, errors.New("missing machine name")
	}

	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	m := &Machine{Name: name, ClusterId: cluster.Id}
	found, err := s.engine.Get(m)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("credential `%s` already exists", name)
	}

	return decodeMachine(m)
}

func (s *machineXormStore) Create(obj *clusterv1.Machine) (*clusterv1.Machine, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing machine")
	} else if obj.Name == "" {
		return nil, errors.New("missing machine name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&Machine{Name: obj.Name, ClusterId: cluster.Id})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if found {
		return nil, errors.Errorf("machine `%s` already exists", obj.Name)
	}

	obj.CreationTimestamp = metav1.Time{Time: time.Now()}
	machine, err := encodeMachine(obj)
	if err != nil {
		return nil, err
	}
	machine.ClusterId = cluster.Id

	_, err = s.engine.Insert(machine)
	return obj, err
}

func (s *machineXormStore) Update(obj *clusterv1.Machine) (*clusterv1.Machine, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing machine")
	} else if obj.Name == "" {
		return nil, errors.New("missing machine name")
	}
	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	err = api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&Machine{Name: obj.Name, ClusterId: cluster.Id})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("machine `%s` not found", obj.Name)
	}

	machine, err := encodeMachine(obj)
	if err != nil {
		return nil, err
	}
	machine.ClusterId = cluster.Id
	_, err = s.engine.Where(`name = ? AND "cluster_id" = ?`, obj.Name, cluster.Id).Update(machine)
	return obj, err
}

func (s *machineXormStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing machine name")
	}
	cluster, err := s.getCluster()
	if err != nil {
		return err
	}
	_, err = s.engine.Delete(&Machine{Name: name, ClusterId: cluster.Id})
	return err
}

func (s *machineXormStore) UpdateStatus(obj *clusterv1.Machine) (*clusterv1.Machine, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing machine")
	} else if obj.Name == "" {
		return nil, errors.New("missing machine name")
	}
	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	err = api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	machine := &Machine{Name: obj.Name, ClusterId: cluster.Id}
	found, err := s.engine.Get(machine)
	if err != nil {
		return nil, errors.Errorf("Machine `%s` does not exist. Reason: %v", obj.Name, err)
	}
	if !found {
		return nil, errors.Errorf("Machine `%s` does not exist", obj.Name)
	}

	existing, err := decodeMachine(machine)
	if err != nil {
		return nil, err
	}
	existing.Status = obj.Status

	updated, err := encodeMachine(existing)
	if err != nil {
		return nil, err
	}
	_, err = s.engine.Where(`name = ? AND "cluster_id" = ?`, obj.Name, cluster.Name, cluster.Id).Update(updated)
	return existing, err
}

func (s *machineXormStore) getCluster() (*Cluster, error) {
	cluster := &Cluster{
		Name:    s.cluster,
		OwnerId: s.owner,
	}
	has, err := s.engine.Get(cluster)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("cluster not exists")
	}
	return cluster, nil
}
