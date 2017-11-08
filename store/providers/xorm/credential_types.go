package xorm

import (
	"encoding/json"
	"time"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Metadata          string     `xorm:"metadata not null 'metadata'"`
	Spec              string     `xorm:"spec not null 'spec'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint deleted 'deletionTimestamp'"`
}

func (Credential) TableName() string {
	return `"pharmer"."credential"`
}

func encodeCredential(in *api.Credential) (*Credential, error) {
	cred := &Credential{
		Kind:              in.Kind,
		APIVersion:        in.APIVersion,
		Name:              in.Name,
		ResourceVersion:   in.ResourceVersion,
		Generation:        in.Generation,
		Spec:              in.Spec.String(),
		CreationTimestamp: in.CreationTimestamp.Time,
		DateModified:      time.Now(),
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

	metadata, err := json.Marshal(in.Spec.Data)
	if err != nil {
		return nil, err
	}
	cred.Metadata = string(metadata)

	return cred, nil
}

func decodeCredential(in *Credential) (*api.Credential, error) {
	var data map[string]string
	if err := json.Unmarshal([]byte(in.Metadata), data); err != nil {
		return nil, err
	}
	var label map[string]string
	if err := json.Unmarshal([]byte(in.Labels), label); err != nil {
		return nil, err
	}
	return &api.Credential{
		TypeMeta: metav1.TypeMeta{
			Kind:       in.Kind,
			APIVersion: in.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              in.Name,
			CreationTimestamp: metav1.Time{Time: in.CreationTimestamp},
		},
		Spec: api.CredentialSpec{
			Provider: label[api.ResourceProviderCredential],
			Data:     data,
		},
	}, nil

}
