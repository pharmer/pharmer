package xorm

import (
	"encoding/json"
	"time"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	Metadata          string     `xorm:"metadata not null 'metadata'"`
	Spec              string     `xorm:"spec not null 'spec'"`
	Status            string     `xorm:"status not null 'status'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint deleted 'deletionTimestamp'"`
}

func (NodeGroup) TableName() string {
	return `"pharmer"."nodegroup"`
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
		Spec:              in.Spec.String(),
		Status:            in.Status.String(),
		CreationTimestamp: in.CreationTimestamp.Time,
		DateModified:      time.Now(),
		DeletionTimestamp: &in.DeletionTimestamp.Time,
	}
	labels, err := json.Marshal(in.ObjectMeta.Labels)
	if err != nil {
		return nil, err
	}
	ng.Labels = string(labels)

	metadata, err := json.Marshal(in.ObjectMeta)
	if err != nil {
		return nil, err
	}
	ng.Metadata = string(metadata)

	return ng, nil
}

func decodeNodeGroup(in *NodeGroup) (*api.NodeGroup, error) {
	var label map[string]string
	if err := json.Unmarshal([]byte(in.Labels), label); err != nil {
		return nil, err
	}
	var spec api.NodeGroupSpec
	if err := json.Unmarshal([]byte(in.Spec), spec); err != nil {
		return nil, err
	}
	var status api.NodeGroupStatus
	if err := json.Unmarshal([]byte(in.Status), status); err != nil {
		return nil, err
	}
	return &api.NodeGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       in.Kind,
			APIVersion: in.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              in.Name,
			UID:               types.UID(in.UID),
			CreationTimestamp: metav1.Time{Time: in.CreationTimestamp},
			DeletionTimestamp: &metav1.Time{Time: *in.DeletionTimestamp},
			Labels:            label,
			ClusterName:       in.ClusterName,
		},
		Spec:   spec,
		Status: status,
	}, nil
}
