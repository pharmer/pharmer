package xorm

import (
	"encoding/json"
	"time"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
)

type Cluster struct {
	Id                int64
	Kind              string     `xorm:"text not null 'kind'"`
	APIVersion        string     `xorm:"text not null 'apiVersion'"`
	Name              string     `xorm:"text not null 'name'"`
	UID               string     `xorm:"text not null 'uid'"`
	ResourceVersion   string     `xorm:"text not null 'resourceVersion'"`
	Generation        int64      `xorm:"bigint not null 'generation'"`
	Labels            string     `xorm:"jsonb not null default '{}' 'labels'"`
	Data              string     `xorm:"text not null 'data'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deletionTimestamp'"`
	OwnerId           string     `xorm:"text  null 'ownerId'"`
}

func (Cluster) TableName() string {
	return `"cluster"`
}

func encodeCluster(in *api.Cluster) (*Cluster, error) {
	cluster := &Cluster{
		Kind:              in.Kind,
		APIVersion:        in.APIVersion,
		Name:              in.Name,
		UID:               string(in.UID),
		ResourceVersion:   in.ResourceVersion,
		Generation:        in.Generation,
		DeletionTimestamp: nil,
	}
	if in.DeletionTimestamp != nil {
		cluster.DeletionTimestamp = &in.DeletionTimestamp.Time
	}
	labels, err := json.Marshal(in.ObjectMeta.Labels)
	if err != nil {
		return nil, err
	}
	cluster.Labels = string(labels)

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
