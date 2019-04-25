package xorm

import (
	"encoding/json"
	"time"

	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
)

type Credential struct {
	Id                int64
	Name              string     `xorm:"text not null 'name'"`
	UID               string     `xorm:"text not null 'uid'"`
	Data              string     `xorm:"text not null 'data'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'created_unix'"`
	DateModified      time.Time  `xorm:"bigint updated 'updated_unix'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deleted_unix'"`
	OwnerId           string     `xorm:"text null 'owner_id'"`
}

func (Credential) TableName() string {
	return `"ac_cluster_credential"`
}

func encodeCredential(in *cloudapi.Credential) (*Credential, error) {
	cred := &Credential{
		//Kind:              in.Kind,
		//APIVersion:        in.APIVersion,
		Name: in.Name,
		//ResourceVersion:   in.ResourceVersion,
		//Generation:        in.Generation,
		DeletionTimestamp: nil,
	}
	/*label := map[string]string{
		api.ResourceProviderCredential: in.Spec.Provider,
	}

	labels, err := json.Marshal(label)
	if err != nil {
		return nil, err
	}
	cred.Labels = string(labels)*/

	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	cred.Data = string(data)

	return cred, nil
}

func decodeCredential(in *Credential) (*cloudapi.Credential, error) {
	var obj cloudapi.Credential
	if err := json.Unmarshal([]byte(in.Data), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}
