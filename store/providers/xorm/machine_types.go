package xorm

import (
	"encoding/json"

	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type Machine struct {
	ID        int64  `xorm:"pk autoincr"`
	Name      string `xorm:"INDEX NOT NULL"`
	Data      string `xorm:"text NOT NULL"`
	ClusterID int64  `xorm:"INDEX NOT NULL"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Machine) TableName() string {
	return "ac_cluster_machine"
}

func encodeMachine(in *clusterapi.Machine) (*Machine, error) {
	machine := &Machine{
		Name:        in.Name,
		CreatedUnix: in.CreationTimestamp.Time.Unix(),
		DeletedUnix: nil,
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
