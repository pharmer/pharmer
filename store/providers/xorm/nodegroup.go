package xorm

import (
	"time"

	"github.com/go-xorm/xorm"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type nodeGroupXormStore struct {
	engine  *xorm.Engine
	cluster string
	owner   string
}

var _ store.NodeGroupStore = &nodeGroupXormStore{}

func (s *nodeGroupXormStore) List(opts metav1.ListOptions) ([]*api.NodeGroup, error) {
	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	result := make([]*api.NodeGroup, 0)
	var nodeGroups []NodeGroup
	err = s.engine.Where(`"clusterId" = ?`, cluster.Id).Find(&nodeGroups)
	if err != nil {
		return nil, err
	}

	for _, ng := range nodeGroups {
		decode, err := decodeNodeGroup(&ng)
		if err != nil {
			return nil, errors.Errorf("failed to list node groups. Reason: %v", err)
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

	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	ng := &NodeGroup{Name: name, ClusterId: cluster.Id, ClusterName: cluster.Name}
	found, err := s.engine.Get(ng)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("credential `%s` already exists", name)
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
	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	err = api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&NodeGroup{Name: obj.Name, ClusterName: cluster.Name, ClusterId: cluster.Id})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if found {
		return nil, errors.Errorf("node group `%s` already exists", obj.Name)
	}

	obj.CreationTimestamp = metav1.Time{Time: time.Now()}
	nodeGroup, err := encodeNodeGroup(obj)
	if err != nil {
		return nil, err
	}
	nodeGroup.ClusterId = cluster.Id

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
	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	err = api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&NodeGroup{Name: obj.Name, ClusterName: cluster.Name, ClusterId: cluster.Id})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("node group `%s` not found", obj.Name)
	}

	ng, err := encodeNodeGroup(obj)
	if err != nil {
		return nil, err
	}
	ng.ClusterId = cluster.Id
	_, err = s.engine.Where(`name = ? AND "clusterName" = ? AND "clusterId" = ?`, obj.Name, cluster.Name, cluster.Id).Update(ng)
	return obj, err
}

func (s *nodeGroupXormStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing node group name")
	}
	cluster, err := s.getCluster()
	if err != nil {
		return err
	}

	_, err = s.engine.Delete(&NodeGroup{Name: name, ClusterName: cluster.Name, ClusterId: cluster.Id})
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
	cluster, err := s.getCluster()
	if err != nil {
		return nil, err
	}

	err = api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	ng := &NodeGroup{Name: obj.Name, ClusterName: cluster.Name, ClusterId: cluster.Id}
	found, err := s.engine.Get(ng)
	if err != nil {
		return nil, errors.Errorf("NodeGroup `%s` does not exist. Reason: %v", obj.Name, err)
	}
	if !found {
		return nil, errors.Errorf("NodeGroup `%s` does not exist", obj.Name)
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
	_, err = s.engine.Where(`name = ? AND "clusterName" = ? AND "clusterId" = ?`, obj.Name, cluster.Name, cluster.Id).Update(updated)
	return existing, err
}

func (s *nodeGroupXormStore) getCluster() (*Cluster, error) {
	cluster := &Cluster{
		Name:    s.cluster,
		OwnerId: s.owner,
	}
	has, err := s.engine.Get(cluster)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("cluster not exists")
	}
	return cluster, nil
}
