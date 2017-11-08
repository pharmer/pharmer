package xorm

import (
	"time"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
)

type Credential struct {
	Id                int64
	Kind              string     `xorm:"text not null 'kind'"`
	APIVersion        string     `xorm:"text not null 'apiVersion'"`
	Name              string     `xorm:"text not null 'name'"`
	ClusterName       string     `xorm:"text not null 'clusterName'"`
	UID               string     `xorm:"text not null 'uid'"`
	ResourceVersion   string     `xorm:"text not null 'resourceVersion'"`
	Generation        int64      `xorm:"bigint not null 'generation'"`
	Labels            string     `xorm:"jsonb not null default '{}' 'labels'"`
	Metadata          string     `xorm:"metadata not null 'data'"`
	Spec              string     `xorm:"spec not null 'data'"`
	Status            string     `xorm:"status not null 'data'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint deleted 'deletionTimestamp'"`
}

func encodeCredential(in *api.Credential) (*Credential, error) {
	return nil, store.ErrNotImplemented
}

func decodeCredential(in *Credential) (*api.Credential, error) {
	return nil, store.ErrNotImplemented
}
