package xorm

import (
	"github.com/go-xorm/xorm"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
)

type sshKeyXormStore struct {
	engine  *xorm.Engine
	cluster string
}

var _ store.SSHKeyStore = &sshKeyXormStore{}

func (s *sshKeyXormStore) Get(name string) ([]byte, []byte, error) {
	if s.cluster == "" {
		return nil, nil, errors.New("missing cluster name")
	}
	if name == "" {
		return nil, nil, errors.New("missing ssh key name")
	}

	sshKey := &SSHKey{
		Name:        name,
		ClusterName: s.cluster,
	}
	found, err := s.engine.Get(sshKey)
	if !found {
		return nil, nil, errors.Errorf("ssh key `%s` for cluster `%s` not found", name, s.cluster)
	}
	if err != nil {
		return nil, nil, err
	}
	return decodeSSHKey(sshKey)
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

	sshKey := &SSHKey{
		Name:        name,
		ClusterName: s.cluster,
	}
	found, err := s.engine.Get(sshKey)
	if found {
		return errors.Errorf("ssh key `%s` for cluster `%s` already exists", name, s.cluster)
	}
	if err != nil {
		return err
	}
	sshKey, err = encodeSSHKey(pubKey, privKey)
	if err != nil {
		return err
	}
	sshKey.Name = name
	sshKey.ClusterName = s.cluster
	sshKey.UID = string(uuid.NewUUID())

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

	_, err := s.engine.Delete(&SSHKey{Name: name, ClusterName: s.cluster})
	return err
}
