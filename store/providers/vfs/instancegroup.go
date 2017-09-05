package vfs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/store"
	"github.com/graymeta/stow"
)

type InstanceGroupFileStore struct {
	container stow.Container
	prefix    string
	cluster   string
}

var _ store.InstanceGroupStore = &InstanceGroupFileStore{}

func (s *InstanceGroupFileStore) resourceHome() string {
	return filepath.Join(s.prefix, "clusters", s.cluster, "instancegroups")
}

func (s *InstanceGroupFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *InstanceGroupFileStore) List(opts api.ListOptions) ([]*api.InstanceGroup, error) {
	result := make([]*api.InstanceGroup, 0)
	cursor := stow.CursorStart
	for {
		page, err := s.container.Browse(s.resourceHome()+"/", string(os.PathSeparator), cursor, pageSize)
		if err != nil {
			return nil, fmt.Errorf("Failed to list instance groups. Reason: %v", err)
		}
		for _, item := range page.Items {
			r, err := item.Open()
			if err != nil {
				return nil, fmt.Errorf("Failed to list instance groups. Reason: %v", err)
			}
			var obj api.InstanceGroup
			err = json.NewDecoder(r).Decode(&obj)
			if err != nil {
				return nil, fmt.Errorf("Failed to list instance groups. Reason: %v", err)
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

func (s *InstanceGroupFileStore) Get(name string) (*api.InstanceGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if name == "" {
		return nil, errors.New("Missing instance group name")
	}

	item, err := s.container.Item(s.resourceID(name))
	if err != nil {
		return nil, fmt.Errorf("InstanceGroup `%s` does not exist. Reason: %v", name, err)
	}

	r, err := item.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var existing api.InstanceGroup
	err = json.NewDecoder(r).Decode(&existing)
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *InstanceGroupFileStore) Create(obj *api.InstanceGroup) (*api.InstanceGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance group")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance group name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)
	_, err = s.container.Item(id)
	if err == nil {
		return nil, fmt.Errorf("InstanceGroup `%s` already exists", obj.Name)
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
	return obj, err
}

func (s *InstanceGroupFileStore) Update(obj *api.InstanceGroup) (*api.InstanceGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance group")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance group name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	_, err = s.container.Item(id)
	if err != nil {
		return nil, fmt.Errorf("InstanceGroup `%s` does not exist. Reason: %v", obj.Name, err)
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
	return obj, err
}

func (s *InstanceGroupFileStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("Missing cluster name")
	}
	if name == "" {
		return errors.New("Missing instance group name")
	}
	return s.container.RemoveItem(s.resourceID(name))
}

func (s *InstanceGroupFileStore) UpdateStatus(obj *api.InstanceGroup) (*api.InstanceGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("Missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("Missing instance group")
	} else if obj.Name == "" {
		return nil, errors.New("Missing instance group name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	item, err := s.container.Item(id)
	if err != nil {
		return nil, fmt.Errorf("InstanceGroup `%s` does not exist. Reason: %v", obj.Name, err)
	}

	r, err := item.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var existing api.InstanceGroup
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
