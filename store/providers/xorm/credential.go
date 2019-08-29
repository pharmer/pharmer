package xorm

import (
	"strconv"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"
)

type credentialXormStore struct {
	engine *xorm.Engine
	owner  int64
}

var _ store.CredentialStore = &credentialXormStore{}

func (s *credentialXormStore) List(opts metav1.ListOptions) ([]*cloudapi.Credential, error) {
	result := make([]*cloudapi.Credential, 0)
	var credentials []Credential
	err := s.engine.Where(`"ownerId"=?`, s.owner).Find(&credentials)
	if err != nil {
		return nil, err
	}
	for _, credential := range credentials {
		decode, err := DecodeCredential(&credential)
		if err != nil {
			return nil, errors.Errorf("failed to list credentials. Reason: %v", err)
		}
		result = append(result, decode)
	}

	return result, nil
}

func (s *credentialXormStore) Get(name string) (*cloudapi.Credential, error) {
	if name == "" {
		return nil, errors.New("missing credential name")
	}
	cred := &Credential{
		Name:    name,
		OwnerID: s.owner,
	}
	if s.owner == -1 {
		id, err := strconv.Atoi(name)
		if err != nil {
			return nil, err
		}
		cred = &Credential{ID: int64(id)}
	}

	found, err := s.engine.Get(cred)
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("credential %s does not exists", name)
	}

	return DecodeCredential(cred)
}

func (s *credentialXormStore) Create(obj *cloudapi.Credential) (*cloudapi.Credential, error) {
	if obj == nil {
		return nil, errors.New("missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}
	found, err := s.engine.Get(&Credential{Name: obj.Name, DeletedUnix: nil, OwnerID: s.owner})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if found {
		return nil, errors.Errorf("credential `%s` already exists", obj.Name)
	}
	obj.CreationTimestamp = metav1.Time{Time: time.Now()}
	cred, err := EncodeCredential(obj)
	if err != nil {
		return nil, err
	}
	cred.UID = string(uuid.NewUUID())
	cred.OwnerID = s.owner
	_, err = s.engine.Insert(cred)

	return obj, err
}

func (s *credentialXormStore) Update(obj *cloudapi.Credential) (*cloudapi.Credential, error) {
	if obj == nil {
		return nil, errors.New("missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&Credential{Name: obj.Name, OwnerID: s.owner})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("credential `%s` does not exist. Reason: %v", obj.Name, err)
	}

	cred, err := EncodeCredential(obj)
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
	_, err := s.engine.Delete(&Credential{Name: name, OwnerID: s.owner})
	return err
}
