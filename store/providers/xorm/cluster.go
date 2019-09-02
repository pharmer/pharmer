package xorm

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
	"gomodules.xyz/secrets/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"
)

type clusterXormStore struct {
	engine *xorm.Engine
	owner  int64
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
		apiCluster := new(api.Cluster)
		if err := json.Unmarshal([]byte(cluster.Data.Data), apiCluster); err != nil {
			log.Error(err, "failed to unmarshal cluster")
			return nil, err
		}
		result = append(result, apiCluster)
	}

	return result, nil
}

// Get() takes clusterName as parameter
// we need to get cluster from apiServer using clusterID
// so for that case, we've to set ownerID to -1
// it should be only used from apiserver
func (s *clusterXormStore) Get(name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("missing cluster name")
	}

	cluster := &Cluster{Name: name, OwnerID: s.owner}
	if s.owner == -1 {
		id, err := strconv.Atoi(name)
		if err != nil {
			return nil, err
		}
		cluster = &Cluster{ID: int64(id)}
	}
	found, err := s.engine.Get(cluster)
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("cluster `%s` does not exists", name)
	}

	apiCluster := new(api.Cluster)
	if err := json.Unmarshal([]byte(cluster.Data.Data), apiCluster); err != nil {
		log.Error(err, "failed to unmarshal cluster")
		return nil, err
	}

	return apiCluster, nil
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

	found, err := s.engine.Get(&Cluster{Name: obj.Name, OwnerID: s.owner})
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if found {
		return nil, errors.Errorf("cluster `%s` already exists", obj.Name)
	}

	obj.CreationTimestamp = metav1.Time{Time: time.Now()}

	data, err := json.Marshal(obj)
	if err != nil {
		log.Error(err, "failed to marshal cluster")
		return nil, err
	}
	cluster := &Cluster{
		OwnerID: s.owner,
		Name:    obj.Name,
		Data: types.SecureString{
			Data: string(data),
		},
		IsPrivate:   false,
		CreatedUnix: obj.CreationTimestamp.Unix(),
		DeletedUnix: nil,
	}

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

	cluster := &Cluster{
		Name:    obj.Name,
		OwnerID: s.owner,
	}
	found, err := s.engine.Get(cluster)
	if err != nil {
		return nil, errors.Errorf("reason: %v", err)
	}
	if !found {
		return nil, errors.Errorf("cluster `%s` does not exists", obj.Name)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		log.Error(err, "failed to marshal cluster")
		return nil, err
	}
	cluster.Data = types.SecureString{
		Data: string(data),
	}

	_, err = s.engine.Where(`name = ?`, obj.Name).Update(cluster)
	return obj, err
}

func (s *clusterXormStore) Delete(name string) error {
	if name == "" {
		return errors.New("missing cluster name")
	}
	_, err := s.engine.Delete(&Cluster{Name: name, OwnerID: s.owner})
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

	cluster := &Cluster{Name: obj.Name, OwnerID: s.owner}
	found, err := s.engine.Get(cluster)
	if err != nil {
		return nil, errors.Errorf("cluster `%s` does not exist. Reason: %v", obj.Name, err)
	}
	if !found {
		return nil, errors.Errorf("cluster `%s` does not exist", obj.Name)
	}

	existing := new(api.Cluster)
	if err := json.Unmarshal([]byte(cluster.Data.Data), existing); err != nil {
		log.Error(err, "failed to unmarshal cluster")
		return nil, err
	}
	existing.Status = obj.Status

	data, err := json.Marshal(existing)
	if err != nil {
		log.Error(err, "failed to marshal cluster")
		return nil, err
	}

	cluster.Data = types.SecureString{
		Data: string(data),
	}

	_, err = s.engine.Where(`name = ?`, obj.Name).Where(`"owner_id"=?`, s.owner).Update(cluster)
	return existing, err
}
