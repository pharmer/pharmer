package xorm

import (
	"encoding/json"
	"time"

	api "github.com/pharmer/pharmer/apis/v1beta1"
)

type NodeGroup struct {
	Id                int64
	Kind              string     `xorm:"text not null 'kind'"`
	APIVersion        string     `xorm:"text not null 'apiVersion'"`
	Name              string     `xorm:"text not null 'name'"`
	ClusterName       string     `xorm:"text not null 'clusterName'"`
	UID               string     `xorm:"text not null 'uid'"`
	ResourceVersion   string     `xorm:"text not null 'resourceVersion'"`
	Generation        int64      `xorm:"bigint not null 'generation'"`
	Labels            string     `xorm:"jsonb not null default '{}' 'labels'"`
	Data              string     `xorm:"text not null 'data'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deletionTimestamp'"`
	ClusterId         int64      `xorm:"bigint not null 'clusterId'"`
}

func (NodeGroup) TableName() string {
	return "cluster_nodegroup"
}

func encodeNodeGroup(in *api.NodeGroup) (*NodeGroup, error) {
	ng := &NodeGroup{
		Kind:              in.Kind,
		APIVersion:        in.APIVersion,
		Name:              in.Name,
		ClusterName:       in.ObjectMeta.ClusterName,
		UID:               string(in.ObjectMeta.UID),
		ResourceVersion:   in.ResourceVersion,
		Generation:        in.Generation,
		CreationTimestamp: in.CreationTimestamp.Time,
		DeletionTimestamp: nil,
	}
	labels, err := json.Marshal(in.ObjectMeta.Labels)
	if err != nil {
		return nil, err
	}
	ng.Labels = string(labels)

	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	ng.Data = string(data)

	return ng, nil
}

func decodeNodeGroup(in *NodeGroup) (*api.NodeGroup, error) {
	var obj api.NodeGroup
	if err := json.Unmarshal([]byte(in.Data), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}
