package xorm

import (
	"encoding/json"

	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
)

type Credential struct {
	ID      int64  `xorm:"pk autoincr"`
	OwnerID int64  `xorm:"UNIQUE(s)"`
	Name    string `xorm:"UNIQUE(s) INDEX NOT NULL"`
	UID     string `xorm:"uid UNIQUE"`
	Data    string `xorm:"text NOT NULL"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Credential) TableName() string {
	return "ac_cluster_credential"
}

func encodeCredential(in *cloudapi.Credential) (*Credential, error) {
	cred := &Credential{
		//Kind:              in.Kind,
		//APIVersion:        in.APIVersion,
		Name: in.Name,
		//ResourceVersion:   in.ResourceVersion,
		//Generation:        in.Generation,
		DeletedUnix: nil,
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
