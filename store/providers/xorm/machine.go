package xorm

import (
	"encoding/json"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
	"gomodules.xyz/secrets/types"
	"gomodules.xyz/secrets/xkms"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type machineXormStore struct {
	engine  *xorm.Engine
	cluster string
	owner   int64
}

var _ store.MachineStore = &machineXormStore{}

func (s *machineXormStore) List(opts metav1.ListOptions) ([]*clusterapi.Machine, error) {
	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	result := make([]*clusterapi.Machine, 0)
	var machines []Machine
	err = s.engine.Where(`"cluster_id" = ?`, cluster.ID).Find(&machines)
	if err != nil {
		return nil, err
	}

	for _, m := range machines {
		apiMachine := new(clusterapi.Machine)
		if err := json.Unmarshal([]byte(m.Data.Data), apiMachine); err != nil {
			log.Error(err, "failed to unmarshal machine")
			return nil, err
		}
		result = append(result, apiMachine)
	}
	return result, nil
}

func (s *machineXormStore) Get(name string) (*clusterapi.Machine, error) {
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

	m := &Machine{Name: name, ClusterID: cluster.ID}
	found, err := s.engine.Get(m)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("credential `%s` already exists", name)
	}

	apiMachine := new(clusterapi.Machine)
	if err := json.Unmarshal([]byte(m.Data.Data), apiMachine); err != nil {
		return nil, err
	}
	return apiMachine, nil
}

func (s *machineXormStore) Create(obj *clusterapi.Machine) (*clusterapi.Machine, error) {
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

	found, err := s.engine.Get(&Machine{Name: obj.Name, ClusterID: cluster.ID})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if found {
		return nil, errors.Errorf("machine `%s` already exists", obj.Name)
	}

	obj.CreationTimestamp = metav1.Time{Time: time.Now()}

	types.Config(xkms.RotateQuarterly)
	data, err := json.Marshal(obj)
	if err != nil {
		log.Error(err, "failed to marshal machine data")
		return nil, err
	}
	machine := &Machine{
		Name: obj.Name,
		Data: types.SecureString{
			Data: string(data),
		},
		ClusterID:   cluster.ID,
		CreatedUnix: obj.CreationTimestamp.Unix(),
		DeletedUnix: nil,
	}

	_, err = s.engine.Insert(machine)
	return obj, err
}

func (s *machineXormStore) Update(obj *clusterapi.Machine) (*clusterapi.Machine, error) {
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

	machine := &Machine{
		Name:      obj.Name,
		ClusterID: cluster.ID,
	}
	found, err := s.engine.Get(machine)
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("machine `%s` not found", obj.Name)
	}

	types.Config(xkms.RotateQuarterly)
	data, err := json.Marshal(obj)
	if err != nil {
		log.Error(err, "failed to marshal machine data")
		return nil, err
	}
	machine.Data.Data = string(data)

	_, err = s.engine.Where(`name = ? AND "cluster_id" = ?`, obj.Name, cluster.ID).Update(machine)
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
	_, err = s.engine.Delete(&Machine{Name: name, ClusterID: cluster.ID})
	return err
}

func (s *machineXormStore) UpdateStatus(obj *clusterapi.Machine) (*clusterapi.Machine, error) {
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

	machine := &Machine{Name: obj.Name, ClusterID: cluster.ID}
	found, err := s.engine.Get(machine)
	if err != nil {
		return nil, errors.Errorf("Machine `%s` does not exist. Reason: %v", obj.Name, err)
	}
	if !found {
		return nil, errors.Errorf("Machine `%s` does not exist", obj.Name)
	}

	existing := new(clusterapi.Machine)
	if err := json.Unmarshal([]byte(machine.Data.Data), existing); err != nil {
		log.Error(err, "failed to unmarshal machine")
		return nil, err
	}
	existing.Status = obj.Status

	types.Config(xkms.RotateQuarterly)
	data, err := json.Marshal(existing)
	if err != nil {
		log.Error(err, "failed to marshal machine")
		return nil, err
	}

	machine.Data.Data = string(data)

	_, err = s.engine.Where(`name = ? AND "cluster_id" = ?`, obj.Name, cluster.ID).Update(machine)
	return existing, err
}

func (s *machineXormStore) getCluster() (*Cluster, error) {
	cluster := &Cluster{
		Name:    s.cluster,
		OwnerID: s.owner,
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
