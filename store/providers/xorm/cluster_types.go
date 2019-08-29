package xorm

import (
	"encoding/json"

	"github.com/appscode/go/types"
	stypes "gomodules.xyz/secrets/types"
	api "pharmer.dev/pharmer/apis/v1alpha1"
)

// Cluster represents a kubernets cluster
type Cluster struct {
	ID        int64  `xorm:"pk autoincr"`
	OwnerID   int64  `xorm:"UNIQUE(s)"`
	Name      string `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Data      []byte `xorm:"blob NOT NULL"`
	IsPrivate bool   `xorm:"INDEX"`
	SecretID  string

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Cluster) TableName() string {
	return "ac_cluster"
}

func EncodeCluster(in *api.Cluster) (*Cluster, error) {
	secretId := stypes.RotateQuarterly()
	data, err := json.Marshal(in)
	if err != nil {
		log.Error(err, "failed to marshal cluster data")
		return nil, err
	}

	cipher, err := encryptData(secretId, data)
	if err != nil {
		log.Error(err, "failed to encrypt cluster data")
		return nil, err
	}

	cluster := &Cluster{
		Name:        in.Name,
		Data:        cipher,
		SecretID:    secretId,
		DeletedUnix: nil,
	}
	if in.DeletionTimestamp != nil {
		cluster.DeletedUnix = types.Int64P(in.DeletionTimestamp.Time.Unix())
	}

	return cluster, nil
}

func DecodeCluster(in *Cluster) (*api.Cluster, error) {
	data, err := decryptData(in.SecretID, in.Data)
	if err != nil {
		log.Error(err, "failed to decrypt cluster data")
		return nil, err
	}
	var obj api.Cluster
	if err := json.Unmarshal(data, &obj); err != nil {
		log.Error(err, "failed to unmarshal cluster data")
		return nil, err
	}
	return &obj, nil
}
