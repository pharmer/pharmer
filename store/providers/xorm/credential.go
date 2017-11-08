package xorm

import (
	"errors"
	"fmt"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/store"
	"github.com/go-xorm/xorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type credentialXormStore struct {
	engine *xorm.Engine
}

var _ store.CredentialStore = &credentialXormStore{}

func (s *credentialXormStore) List(opts metav1.ListOptions) ([]*api.Credential, error) {
	result := make([]*api.Credential, 0)
	var credentials []Credential
	err := s.engine.Find(credentials)
	if err != nil {
		return nil, err
	}
	for _, credential := range credentials {
		decode, err := decodeCredential(&credential)
		if err != nil {
			return nil, fmt.Errorf("failed to list credentials. Reason: %v", err)
		}
		result = append(result, decode)
	}

	return result, nil
}

func (s *credentialXormStore) Get(name string) (*api.Credential, error) {
	if name == "" {
		return nil, errors.New("missing credential name")
	}
	cred := &Credential{
		Name: name,
	}

	found, err := s.engine.Get(cred)
	if !found {
		return nil, fmt.Errorf("credential %s does not exists", name)
	}
	if err != nil {
		return nil, fmt.Errorf("reason: %v", err)
	}

	return decodeCredential(cred)
}

func (s *credentialXormStore) Create(obj *api.Credential) (*api.Credential, error) {
	if obj == nil {
		return nil, errors.New("missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}
	found, err := s.engine.Get(&Credential{Name: obj.Name})
	if found {
		return nil, fmt.Errorf("credential `%s` already exists", obj.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("reason: %v", err)
	}
	cred, err := encodeCredential(obj)
	if err != nil {
		return nil, err
	}
	cred.UID = string(phid.NewCloudCredential())

	_, err = s.engine.Insert(cred)

	return obj, err
}

func (s *credentialXormStore) Update(obj *api.Credential) (*api.Credential, error) {
	if obj == nil {
		return nil, errors.New("missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&Credential{Name: obj.Name})
	if !found {
		return nil, fmt.Errorf("credential `%s` does not exist. Reason: %v", obj.Name, err)
	}
	if err != nil {
		return nil, err
	}

	cred, err := encodeCredential(obj)
	if err != nil {
		return nil, err
	}
	_, err = s.engine.Where(`name = ?`, cred.Name).Update(cred)
	return obj, err
}

func (s *credentialXormStore) Delete(name string) error {
	if name == "" {
		return errors.New("missing credential name")
	}
	_, err := s.engine.Delete(&Credential{Name: name})
	return err
}
