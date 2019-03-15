package xorm

import (
	"time"
)

type SSHKey struct {
	Id                int64
	Name              string     `xorm:"text not null 'name'"`
	ClusterName       string     `xorm:"text not null 'cluster_name'"`
	UID               string     `xorm:"text not null 'uid'"`
	PublicKey         string     `xorm:"string  not null 'public_key'"`
	PrivateKey        string     `xorm:"string  not null 'private_key'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'created_unix'"`
	DateModified      time.Time  `xorm:"bigint updated 'updated_unix'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deleted_unix'"`
	ClusterId         int64      `xorm:"bigint not null 'cluster_id'"`
}

func (SSHKey) TableName() string {
	return `"cluster_ssh_key"`
}

func encodeSSHKey(pub, priv []byte) (*SSHKey, error) {
	return &SSHKey{
		PublicKey:         string(pub),
		PrivateKey:        string(priv),
		DeletionTimestamp: nil,
	}, nil
}

func decodeSSHKey(in *SSHKey) ([]byte, []byte, error) {
	return []byte(in.PublicKey), []byte(in.PrivateKey), nil
}
