package xorm

import (
	"encoding/json"
	"time"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
)

type Credential struct {
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
	OwnerId           string     `xorm:"text null 'ownerId'"`
}

func (Credential) TableName() string {
	return `"cluster_credential"`
}

func encodeCredential(in *api.Credential) (*Credential, error) {
	cred := &Credential{
		Kind:              in.Kind,
		APIVersion:        in.APIVersion,
		Name:              in.Name,
		ResourceVersion:   in.ResourceVersion,
		Generation:        in.Generation,
		DeletionTimestamp: nil,
	}
	label := map[string]string{
		api.ResourceProviderCredential: in.Spec.Provider,
	}

	labels, err := json.Marshal(label)
	if err != nil {
		return nil, err
	}
	cred.Labels = string(labels)

	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	cred.Data = string(data)

	return cred, nil
}

func decodeCredential(in *Credential) (*api.Credential, error) {
	var obj api.Credential
	if err := json.Unmarshal([]byte(in.Data), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}
