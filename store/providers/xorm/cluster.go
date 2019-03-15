package xorm

import (
	"strconv"
	"time"

	"github.com/go-xorm/xorm"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterXormStore struct {
	engine *xorm.Engine
	owner  string
}

var _ store.ClusterStore = &clusterXormStore{}

func (s *clusterXormStore) List(opts metav1.ListOptions) ([]*api.Cluster, error) {
	result := make([]*api.Cluster, 0)
	var clusters []Cluster

	err := s.engine.Where(`"owner_id"=?`, s.owner).Find(&clusters)
	if err != nil {
		return nil, err
	}

	for _, cluster := range clusters {
		decode, err := decodeCluster(&cluster)
		if err != nil {
			return nil, errors.Errorf("failed to list clusters. Reason: %v", err)
		}
		result = append(result, decode)
	}

	return result, nil
}

func (s *clusterXormStore) Get(name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("missing cluster name")
	}

	cluster := &Cluster{Name: name, OwnerId: s.owner}
	if s.owner == "" {
		id, err := strconv.Atoi(name)
		if err != nil {
			return nil, err
		}
		cluster = &Cluster{Id: int64(id)}
	}
	found, err := s.engine.Get(cluster)
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("cluster `%s` does not exists", name)
	}
	return decodeCluster(cluster)
}

func (s *clusterXormStore) Create(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&Cluster{Name: obj.Name, OwnerId: s.owner})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if found {
		return nil, errors.Errorf("cluster `%s` already exists", obj.Name)
	}

	obj.CreationTimestamp = metav1.Time{Time: time.Now()}
	cluster, err := encodeCluster(obj)
	if err != nil {
		return nil, err
	}
	cluster.OwnerId = s.owner
	_, err = s.engine.Insert(cluster)
	return obj, err
}

func (s *clusterXormStore) Update(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	found, err := s.engine.Get(&Cluster{Name: obj.Name, OwnerId: s.owner})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("cluster `%s` does not exists", obj.Name)
	}

	cluster, err := encodeCluster(obj)
	if err != nil {
		return nil, err
	}

	_, err = s.engine.Where(`name = ?`, obj.Name).Update(cluster)
	return obj, err
}

func (s *clusterXormStore) Delete(name string) error {
	if name == "" {
		return errors.New("missing cluster name")
	}
	_, err := s.engine.Delete(&Cluster{Name: name, OwnerId: s.owner})
	return err
}

func (s *clusterXormStore) UpdateStatus(obj *api.Cluster) (*api.Cluster, error) {
	if obj == nil {
		return nil, errors.New("missing cluster")
	} else if obj.Name == "" {
		return nil, errors.New("missing cluster name")
	}
	err := api.AssignTypeKind(obj)
	if err != nil {
		return nil, err
	}

	cluster := &Cluster{Name: obj.Name, OwnerId: s.owner}
	found, err := s.engine.Get(cluster)
	if err != nil {
		return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", obj.Name, err)
	}
	if !found {
		return nil, errors.Errorf("cluster `%s` does not exist", obj.Name)
	}
	existing, err := decodeCluster(cluster)
	if err != nil {
		return nil, err
	}
	existing.Status = obj.Status

	updated, err := encodeCluster(existing)
	if err != nil {
		return nil, err
	}
	_, err = s.engine.Where(`name = ?`, obj.Name).Where(`"owner_id"=?`, s.owner).Update(updated)
	return existing, err
}
