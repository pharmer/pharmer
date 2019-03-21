package xorm

import (
	"encoding/json"
	"time"

	api "github.com/pharmer/pharmer/apis/v1beta1"
)

type Cluster struct {
	Id                int64
	Name              string     `xorm:"text not null 'name'"`
	Data              string     `xorm:"text not null 'data'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'created_unix'"`
	DateModified      time.Time  `xorm:"bigint updated 'updated_unix'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deleted_unix'"`
	OwnerId           string     `xorm:"text  null 'owner_id'"`
	IsPrivate         bool       `xorm:"boolean 'is_private'"`
}

func (Cluster) TableName() string {
	return `"ac_cluster"`
}

func encodeCluster(in *api.Cluster) (*Cluster, error) {
	cluster := &Cluster{
		//Kind:              in.Kind,
		//APIVersion:        in.APIVersion,
		Name: in.Name,
		//UID:               string(in.UID),
		//ResourceVersion:   in.ResourceVersion,
		//Generation:        in.Generation,
		DeletionTimestamp: nil,
	}
	if in.DeletionTimestamp != nil {
		cluster.DeletionTimestamp = &in.DeletionTimestamp.Time
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
