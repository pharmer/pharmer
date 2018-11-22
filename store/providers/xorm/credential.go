package xorm

import (
	"strconv"
	"time"

	"github.com/go-xorm/xorm"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

type credentialXormStore struct {
	engine *xorm.Engine
	owner  string
}

var _ store.CredentialStore = &credentialXormStore{}

func (s *credentialXormStore) List(opts metav1.ListOptions) ([]*api.Credential, error) {
	result := make([]*api.Credential, 0)
	var credentials []Credential
	err := s.engine.Where(`"ownerId"=?`, s.owner).Find(&credentials)
	if err != nil {
		return nil, err
	}
	for _, credential := range credentials {
		decode, err := decodeCredential(&credential)
		if err != nil {
			return nil, errors.Errorf("failed to list credentials. Reason: %v", err)
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
		Name:    name,
		OwnerId: s.owner,
	}
	if s.owner == "" {
		id, err := strconv.Atoi(name)
		if err != nil {
			return nil, err
		}
		cred = &Credential{Id: int64(id)}
	}

	found, err := s.engine.Get(cred)
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("credential %s does not exists", name)
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
	found, err := s.engine.Get(&Credential{Name: obj.Name, DeletionTimestamp: nil, OwnerId: s.owner})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if found {
		return nil, errors.Errorf("credential `%s` already exists", obj.Name)
	}
	obj.CreationTimestamp = metav1.Time{Time: time.Now()}
	cred, err := encodeCredential(obj)
	if err != nil {
		return nil, err
	}
	cred.UID = string(uuid.NewUUID())
	cred.OwnerId = s.owner
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

	found, err := s.engine.Get(&Credential{Name: obj.Name, OwnerId: s.owner})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("credential `%s` does not exist. Reason: %v", obj.Name, err)
	}

	cred, err := encodeCredential(obj)
	if err != nil {
		return nil, err
	}
	_, err = s.engine.Where(`name = ?`, cred.Name).Where(`"ownerId"=?`, s.owner).Update(cred)
	return obj, err
}

func (s *credentialXormStore) Delete(name string) error {
	if name == "" {
		return errors.New("missing credential name")
	}
	_, err := s.engine.Delete(&Credential{Name: name, OwnerId: s.owner})
	return err
}
