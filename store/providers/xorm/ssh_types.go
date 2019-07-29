package xorm

import (
	"time"
)

type SSHKey struct {
	ID          int64
	Name        string `xorm:"text not null 'name'"`
	ClusterName string `xorm:"text not null 'cluster_name'"`
	ClusterID   int64  `xorm:"bigint not null 'cluster_id'"`
	UID         string `xorm:"text not null 'uid'"`
	PublicKey   string `xorm:"string  not null 'public_key'"`
	PrivateKey  string `xorm:"string  not null 'private_key'"`

	CreationTimestamp time.Time  `xorm:"bigint created 'created_unix'"`
	DateModified      time.Time  `xorm:"bigint updated 'updated_unix'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deleted_unix'"`
}

func (SSHKey) TableName() string {
	return "ac_cluster_ssh"
}

func encodeSSHKey(pub, priv []byte) *SSHKey {
	return &SSHKey{
		PublicKey:         string(pub),
		PrivateKey:        string(priv),
		DeletionTimestamp: nil,
	}
}

func decodeSSHKey(in *SSHKey) ([]byte, []byte, error) {
	return []byte(in.PublicKey), []byte(in.PrivateKey), nil
}
