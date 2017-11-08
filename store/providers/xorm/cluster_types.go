package xorm

import (
	"encoding/json"
	"time"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	Metadata          string     `xorm:"metadata not null 'metadata'"`
	Spec              string     `xorm:"spec not null 'spec'"`
	Status            string     `xorm:"status not null 'status'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint deleted 'deletionTimestamp'"`
}

func (Cluster) TableName() string {
	return `"pharmer"."cluster"`
}

func encodeCluster(in *api.Cluster) (*Cluster, error) {
	cluster := &Cluster{
		Kind:              in.Kind,
		APIVersion:        in.APIVersion,
		Name:              in.Name,
		UID:               string(in.UID),
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
	cluster.Labels = string(labels)

	metadata, err := json.Marshal(in.ObjectMeta)
	if err != nil {
		return nil, err
	}
	cluster.Metadata = string(metadata)

	return cluster, nil
}

func decodeCluster(in *Cluster) (*api.Cluster, error) {
	var label map[string]string
	if err := json.Unmarshal([]byte(in.Labels), label); err != nil {
		return nil, err
	}
	var spec api.ClusterSpec
	if err := json.Unmarshal([]byte(in.Spec), spec); err != nil {
		return nil, err
	}
	var status api.ClusterStatus
	if err := json.Unmarshal([]byte(in.Status), status); err != nil {
		return nil, err
	}
	return &api.Cluster{
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
		},
		Spec:   spec,
		Status: status,
	}, nil
}
