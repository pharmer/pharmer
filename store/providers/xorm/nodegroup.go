package xorm

import (
	"errors"
	"fmt"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/store"
	"github.com/go-xorm/xorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type nodeGroupXormStore struct {
	engine  *xorm.Engine
	cluster string
}

var _ store.NodeGroupStore = &nodeGroupXormStore{}

func (s *nodeGroupXormStore) List(opts metav1.ListOptions) ([]*api.NodeGroup, error) {
	result := make([]*api.NodeGroup, 0)
	var nodeGroups []NodeGroup
	err := s.engine.Where(`"clusterName" = ?`, s.cluster).Find(&nodeGroups)
	if err != nil {
		return nil, err
	}

	for _, ng := range nodeGroups {
		decode, err := decodeNodeGroup(&ng)
		if err != nil {
			return nil, fmt.Errorf("failed to list node groups. Reason: %v", err)
		}
		result = append(result, decode)
	}
	return result, nil
}

func (s *nodeGroupXormStore) Get(name string) (*api.NodeGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, errors.New("missing node group name")
	}

	ng := &NodeGroup{Name: name, ClusterName: s.cluster}
	found, err := s.engine.Get(ng)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("credential `%s` already exists", name)
	}

	return decodeNodeGroup(ng)
}

func (s *nodeGroupXormStore) Create(obj *api.NodeGroup) (*api.NodeGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing node group")
	} else if obj.Name == "" {
		return nil, errors.New("missing node group name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&NodeGroup{Name: obj.Name, ClusterName: s.cluster})
	if err != nil {
		return nil, fmt.Errorf("reason: %v", err)
	}
	if found {
		return nil, fmt.Errorf("node group `%s` already exists", obj.Name)
	}

	nodeGroup, err := encodeNodeGroup(obj)
	if err != nil {
		return nil, err
	}

	_, err = s.engine.Insert(nodeGroup)
	return obj, err
}

func (s *nodeGroupXormStore) Update(obj *api.NodeGroup) (*api.NodeGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing node group")
	} else if obj.Name == "" {
		return nil, errors.New("missing node group name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&NodeGroup{Name: obj.Name, ClusterName: s.cluster})
	if !found {
		return nil, fmt.Errorf("node group `%s` not found", obj.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("reason: %v", err)
	}

	ng, err := encodeNodeGroup(obj)
	if err != nil {
		return nil, err
	}
	_, err = s.engine.Where(`name = ? AND "clusterName" = ?`, obj.Name, s.cluster).Update(ng)
	return obj, err
}

func (s *nodeGroupXormStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing node group name")
	}
	_, err := s.engine.Delete(&NodeGroup{Name: name, ClusterName: s.cluster})
	return err
}

func (s *nodeGroupXormStore) UpdateStatus(obj *api.NodeGroup) (*api.NodeGroup, error) {
	if s.cluster == "" {
		return nil, errors.New("missing cluster name")
	}
	if obj == nil {
		return nil, errors.New("missing node group")
	} else if obj.Name == "" {
		return nil, errors.New("missing node group name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	ng := &NodeGroup{Name: obj.Name, ClusterName: s.cluster}
	found, err := s.engine.Get(ng)
	if err != nil {
		return nil, fmt.Errorf("NodeGroup `%s` does not exist. Reason: %v", obj.Name, err)
	}
	if !found {
		return nil, fmt.Errorf("NodeGroup `%s` does not exist", obj.Name)
	}

	existing, err := decodeNodeGroup(ng)
	if err != nil {
		return nil, err
	}
	existing.Status = obj.Status

	updated, err := encodeNodeGroup(existing)
	if err != nil {
		return nil, err
	}
	_, err = s.engine.Where(`name = ? AND "clusterName" = ?`, obj.Name, s.cluster).Update(updated)
	return existing, err
}
