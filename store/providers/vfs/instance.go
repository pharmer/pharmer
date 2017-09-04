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
)

type InstanceFileStore struct {
	container stow.Container
	prefix    string
	cluster   string
}

var _ store.InstanceStore = &InstanceFileStore{}

func (s *InstanceFileStore) resourceHome() string {
	return filepath.Join(s.prefix, "clusters", s.cluster, "instances")
}

func (s *InstanceFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *InstanceFileStore) List(opts api.ListOptions) ([]*api.Instance, error) {
	result := make([]*api.Instance, 0)
	cursor := stow.CursorStart
	for {
		items, nc, err := s.container.Items(s.resourceHome(), cursor, pageSize)
		if err != nil {
			return nil, fmt.Errorf("Failed to list instances. Reason: %v", err)
		}
		for _, item := range items {
			r, err := item.Open()
			if err != nil {
				return nil, fmt.Errorf("Failed to list instances. Reason: %v", err)
			}
			var obj api.Instance
			err = json.NewDecoder(r).Decode(&obj)
			if err != nil {
				return nil, fmt.Errorf("Failed to list instances. Reason: %v", err)
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

func (s *InstanceFileStore) Get(name string) (*api.Instance, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if name == "" {
		return nil, errors.New("Missing instance name")
	}

	item, err := s.container.Item(s.resourceID(name))
	if err != nil {
		return nil, fmt.Errorf("Instance `%s` does not exist. Reason: %v", name, err)
	}

	r, err := item.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var existing api.Instance
	err = json.NewDecoder(r).Decode(&existing)
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *InstanceFileStore) Create(obj *api.Instance) (*api.Instance, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	_, err = s.container.Item(id)
	if err == nil {
		return nil, fmt.Errorf("Instance `%s` already exists", obj.Name)
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
	return obj, err
}

func (s *InstanceFileStore) Update(obj *api.Instance) (*api.Instance, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	_, err = s.container.Item(id)
	if err != nil {
		return nil, fmt.Errorf("Instance `%s` does not exist. Reason: %v", obj.Name, err)
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
	return obj, err
}

func (s *InstanceFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("Missing cluster name")
	}
	if name == "" {
		return errors.New("Missing instance name")
	}
	return s.container.RemoveItem(s.resourceID(name))
}

func (s *InstanceFileStore) UpdateStatus(obj *api.Instance) (*api.Instance, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	item, err := s.container.Item(id)
	if err != nil {
		return nil, fmt.Errorf("Instance `%s` does not exist. Reason: %v", obj.Name, err)
	}

	r, err := item.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var existing api.Instance
	err = json.NewDecoder(r).Decode(&existing)
	if err != nil {
		return nil, err
	}
	existing.Status = obj.Status

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
	return &existing, err
}
