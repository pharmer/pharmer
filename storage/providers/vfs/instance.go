package vfs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/storage"
	"github.com/graymeta/stow"
)

type InstanceFileStore struct {
	container stow.Container
	prefix    string
	cluster   string
}

var _ storage.InstanceStore = &InstanceFileStore{}

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
		page, err := s.container.Browse(s.resourceHome(), "/", cursor, pageSize)
		if err != nil {
			return nil, fmt.Errorf("Failed to list instances. Reason: %v", err)
		}
		for _, item := range page.Items {
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
		cursor = page.Cursor
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

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(obj)
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, &buf, int64(buf.Len()), nil)
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

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(obj)
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, &buf, int64(buf.Len()), nil)
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

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(&existing)
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, &buf, int64(buf.Len()), nil)
	return &existing, err
}

// Deprecated
func (s *InstanceFileStore) SaveInstances(instances []*api.Instance) error {
	for _, instance := range instances {
		if _, err := s.Get(instance.Name); err != nil {
			_, err = s.Create(instance)
			if err != nil {
				return err
			}
		} else {
			_, err = s.Update(instance)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
