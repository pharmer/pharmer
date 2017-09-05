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
	"github.com/tamalsaha/go-oneliners"
)

type ClusterFileStore struct {
	container stow.Container
	prefix    string
}

var _ store.ClusterStore = &ClusterFileStore{}

func (s *ClusterFileStore) resourceHome() string {
	return filepath.Join(s.prefix, "clusters")
}

func (s *ClusterFileStore) resourceID(name string) string {
	return filepath.Join(s.resourceHome(), name+".json")
}

func (s *ClusterFileStore) List(opts api.ListOptions) ([]*api.Cluster, error) {
	result := make([]*api.Cluster, 0)
	cursor := stow.CursorStart
	for {
		page, err := s.container.Browse(s.resourceHome()+"/", string(os.PathSeparator), cursor, pageSize)
		if err != nil {
			return nil, fmt.Errorf("Failed to list clusters. Reason: %v", err)
		}
		for _, item := range page.Items {
			oneliners.FILE(item.Name())
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

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
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

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
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

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return nil, err
	}
	_, err = s.container.Put(id, bytes.NewBuffer(data), int64(len(data)), nil)
	return &existing, err
}
