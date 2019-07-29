package xorm

import (
	"encoding/json"

	"github.com/appscode/go/types"
	api "pharmer.dev/pharmer/apis/v1alpha1"
)

// Cluster represents a kubernets cluster
type Cluster struct {
	ID        int64  `xorm:"pk autoincr"`
	OwnerID   int64  `xorm:"UNIQUE(s)"`
	Name      string `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Data      string `xorm:"text 'data'"`
	IsPrivate bool   `xorm:"INDEX"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Cluster) TableName() string {
	return "ac_cluster"
}

func encodeCluster(in *api.Cluster) (*Cluster, error) {
	cluster := &Cluster{
		//Kind:              in.Kind,
		//APIVersion:        in.APIVersion,
		Name: in.Name,
		//UID:               string(in.UID),
		//ResourceVersion:   in.ResourceVersion,
		//Generation:        in.Generation,
		DeletedUnix: nil,
	}
	if in.DeletionTimestamp != nil {
		cluster.DeletedUnix = types.Int64P(in.DeletionTimestamp.Time.Unix())
	}
	/*labels, err := json.Marshal(in.ObjectMeta.Labels)
	if err != nil {
		return nil, err
	}*/
	//cluster.Labels = string(labels)

	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	cluster.Data = string(data)

	return cluster, nil
}

func decodeCluster(in *Cluster) (*api.Cluster, error) {
	var obj api.Cluster
	if err := json.Unmarshal([]byte(in.Data), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}
