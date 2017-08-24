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

type ClusterFileStore struct {
	container stow.Container
}

var _ storage.ClusterStore = &ClusterFileStore{}

func (s *ClusterFileStore) resourceHome() string {
	return "clusters"
}

func (s *ClusterFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *ClusterFileStore) List(opts api.ListOptions) ([]*api.Cluster, error) {
	result := make([]*api.Cluster, 0)
	cursor := stow.CursorStart
	for {
		page, err := s.container.Browse(s.resourceHome(), "/", cursor, pageSize)
		if err != nil {
			return nil, fmt.Errorf("Failed to list clusters. Reason: %v", err)
		}
		for _, item := range page.Items {
			r, err := item.Open()
			if err != nil {
				return nil, fmt.Errorf("Failed to list clusters. Reason: %v", err)
			}
			var obj api.Cluster
			err = json.NewDecoder(r).Decode(&obj)
			if err != nil {
				return nil, fmt.Errorf("Failed to list clusters. Reason: %v", err)
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

func (s *ClusterFileStore) Get(name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("Missing cluster name")
	}

	item, err := s.container.Item(s.resourceID(name))
	if err != nil {
		return nil, fmt.Errorf("Cluster `%s` does not exist. Reason: %v", name, err)
	}

	r, err := item.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var existing api.Cluster
	err = json.NewDecoder(r).Decode(&existing)
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *ClusterFileStore) Create(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("Missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("Missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	_, err = s.container.Item(id)
	if err == nil {
		return nil, fmt.Errorf("Cluster `%s` already exists", obj.Name)
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(obj)
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, &buf, int64(buf.Len()), nil)
	return obj, err
}

func (s *ClusterFileStore) Update(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("Missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("Missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	_, err = s.container.Item(id)
	if err != nil {
		return nil, fmt.Errorf("Cluster `%s` does not exist. Reason: %v", obj.Name, err)
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(obj)
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, &buf, int64(buf.Len()), nil)
	return obj, err
}

func (s *ClusterFileStore) Delete(name string) error {
	if name == "" {
		return errors.New("Missing cluster name")
	}
	return s.container.RemoveItem(s.resourceID(name))
}

func (s *ClusterFileStore) UpdateStatus(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("Missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("Missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	id := s.resourceID(obj.Name)

	item, err := s.container.Item(id)
	if err != nil {
		return nil, fmt.Errorf("Cluster `%s` does not exist. Reason: %v", obj.Name, err)
	}

	r, err := item.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var existing api.Cluster
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
	return obj, err
}
