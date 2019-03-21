package xorm

import (
	"encoding/json"
	"time"

	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type Machine struct {
	Id                int64
	Name              string     `xorm:"text not null 'name'"`
	Data              string     `xorm:"text not null 'data'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'created_unix'"`
	DateModified      time.Time  `xorm:"bigint updated 'updated_unix'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deleted_unix'"`
	ClusterId         int64      `xorm:"bigint not null 'cluster_id'"`
}

func (Machine) TableName() string {
	return `"ac_cluster_machine"`
}

func encodeMachine(in *clusterapi.Machine) (*Machine, error) {
	machine := &Machine{
		Name:              in.Name,
		CreationTimestamp: in.CreationTimestamp.Time,
		DeletionTimestamp: nil,
	}

	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	machine.Data = string(data)

	return machine, nil
}

func decodeMachine(in *Machine) (*clusterapi.Machine, error) {
	var obj clusterapi.Machine
	if err := json.Unmarshal([]byte(in.Data), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}
