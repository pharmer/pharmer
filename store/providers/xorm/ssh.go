/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package xorm

import (
	"time"

	"pharmer.dev/pharmer/store"

	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
	"gomodules.xyz/secrets/types"
	"k8s.io/apimachinery/pkg/util/uuid"
)

type sshKeyXormStore struct {
	engine  *xorm.Engine
	cluster string
	owner   int64
}

var _ store.SSHKeyStore = &sshKeyXormStore{}

func (s *sshKeyXormStore) Get(name string) ([]byte, []byte, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("missing ssh key name")
	}
	cluster, err := s.getCluster()
	if err != nil {
		return nil, nil, err
	}

	sshKey := &SSHKey{
		Name:        name,
		ClusterName: cluster.Name,
		ClusterID:   cluster.ID,
	}
	found, err := s.engine.Get(sshKey)
	if !found {
		return nil, nil, errors.Errorf("ssh key `%s` for cluster `%s` not found", name, s.cluster)
	}
	if err != nil {
		return nil, nil, err
	}

	return []byte(sshKey.PublicKey), []byte(sshKey.PrivateKey.Data), nil
}

func (s *sshKeyXormStore) Create(name string, pubKey, privKey []byte) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if len(pubKey) == 0 {
		return errors.New("empty ssh public key")
	} else if len(privKey) == 0 {
		return errors.New("empty ssh private key")
	}

	cluster, err := s.getCluster()
	if err != nil {
		return err
	}

	sshKey := &SSHKey{
		Name:        name,
		ClusterName: cluster.Name,
		ClusterID:   cluster.ID,
	}
	found, err := s.engine.Get(sshKey)
	if found {
		return errors.Errorf("ssh key `%s` for cluster `%s` already exists", name, s.cluster)
	}
	if err != nil {
		return err
	}

	sshKey = &SSHKey{
		Name:        name,
		ClusterID:   cluster.ID,
		ClusterName: cluster.Name,
		UID:         string(uuid.NewUUID()),
		PublicKey:   string(pubKey),
		PrivateKey: types.SecureString{
			Data: string(privKey),
		},
		CreatedUnix: time.Now().Unix(),
		DeletedUnix: nil,
	}

	_, err = s.engine.Insert(sshKey)
	return err
}

func (s *sshKeyXormStore) Delete(name string) error {
	if s.cluster == "" {
		return errors.New("missing cluster name")
	}
	if name == "" {
		return errors.New("missing ssh key name")
	}
	cluster, err := s.getCluster()
	if err != nil {
		return err
	}

	_, err = s.engine.Delete(&SSHKey{Name: name, ClusterName: cluster.Name, ClusterID: cluster.ID})
	return err
}

func (s *sshKeyXormStore) getCluster() (*Cluster, error) {
	cluster := &Cluster{
		Name:    s.cluster,
		OwnerID: s.owner,
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
