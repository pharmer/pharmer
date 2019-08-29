package xorm

import (
	"encoding/json"

	"gomodules.xyz/secrets/types"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type Machine struct {
	ID        int64  `xorm:"pk autoincr"`
	Name      string `xorm:"INDEX NOT NULL"`
	Data      []byte `xorm:"blob NOT NULL"`
	ClusterID int64  `xorm:"INDEX NOT NULL"`
	SecretID  string

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Machine) TableName() string {
	return "ac_cluster_machine"
}

func EncodeMachine(in *clusterapi.Machine) (*Machine, error) {
	secretId := types.RotateQuarterly()
	data, err := json.Marshal(in)
	if err != nil {
		log.Error(err, "failed to marshal machine data")
		return nil, err
	}

	cipher, err := encryptData(secretId, data)
	if err != nil {
		log.Error(err, "failed to encrypt machine data")
		return nil, err
	}

	return &Machine{
		Name:        in.Name,
		Data:        cipher,
		SecretID:    secretId,
		CreatedUnix: in.CreationTimestamp.Time.Unix(),
		DeletedUnix: nil,
	}, nil
}

func DecodeMachine(in *Machine) (*clusterapi.Machine, error) {
	data, err := decryptData(in.SecretID, in.Data)
	if err != nil {
		log.Error(err, "failed to decrypt machine data")
		return nil, err
	}

	var obj clusterapi.Machine
	if err := json.Unmarshal(data, &obj); err != nil {
		log.Error(err, "failed to unmarshal machine data")
		return nil, err
	}
	return &obj, nil
}
