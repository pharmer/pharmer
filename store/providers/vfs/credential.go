package vfs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/store"
	"github.com/graymeta/stow"
	"github.com/tamalsaha/go-oneliners"
)

type CredentialFileStore struct {
	container stow.Container
	prefix    string
}

var _ store.CredentialStore = &CredentialFileStore{}

func (s *CredentialFileStore) resourceHome() string {
	return filepath.Join(s.prefix, "credentials")
}

func (s *CredentialFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *CredentialFileStore) List(opts api.ListOptions) ([]*api.Credential, error) {
	result := make([]*api.Credential, 0)
	cursor := stow.CursorStart
	for {
		items, nc, err := s.container.Items(s.resourceHome(), cursor, pageSize)
		if err != nil {
			return nil, fmt.Errorf("Failed to list credentials. Reason: %v", err)
		}
		for _, item := range items {
			r, err := item.Open()
			if err != nil {
				return nil, fmt.Errorf("Failed to list credentials. Reason: %v", err)
			}
			var obj api.Credential
			err = json.NewDecoder(r).Decode(&obj)
			if err != nil {
				return nil, fmt.Errorf("Failed to list credentials. Reason: %v", err)
			}
			result = append(result, &obj)
			r.Close()
		}
		cursor = nc
		if stow.IsCursorEnd(cursor) {
			break
		}
	}
	return result, nil
}

func (s *CredentialFileStore) Get(name string) (*api.Credential, error) {
	if name == "" {
		return nil, errors.New("Missing credential name")
	}
	oneliners.FILE(s.container.ID(), s.resourceID(name))
	item, err := s.container.Item(s.resourceID(name))
	if err != nil {
		return nil, fmt.Errorf("Credential `%s` does not exist. Reason: %v", name, err)
	}

	r, err := item.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var existing api.Credential
	err = json.NewDecoder(r).Decode(&existing)
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *CredentialFileStore) Create(obj *api.Credential) (*api.Credential, error) {
	if obj == nil {
		return nil, errors.New("Missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("Missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	_, err = s.container.Item(id)
	if err == nil {
		return nil, fmt.Errorf("Credential `%s` already exists", obj.Name)
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
	return obj, err
}

func (s *CredentialFileStore) Update(obj *api.Credential) (*api.Credential, error) {
	if obj == nil {
		return nil, errors.New("Missing credential")
	} else if obj.Name == "" {
		return nil, errors.New("Missing credential name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	_, err = s.container.Item(id)
	if err != nil {
		return nil, fmt.Errorf("Credential `%s` does not exist. Reason: %v", obj.Name, err)
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
	return obj, err
}

func (s *CredentialFileStore) Delete(name string) error {
	if name == "" {
		return errors.New("Missing credential name")
	}
	return s.container.RemoveItem(s.resourceID(name))
}
