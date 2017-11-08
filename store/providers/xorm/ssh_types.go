package xorm

import (
	"time"

	"github.com/appscode/pharmer/store"
)

type SSHKey struct {
	Id                int64
	Name              string     `xorm:"text not null 'name'"`
	UID               string     `xorm:"text not null 'uid'"`
	PublicKey         string     `xorm:"string  not null 'publicKey'"`
	PrivateKey        string     `xorm:"string  not null 'privateKey'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint deleted 'deletionTimestamp'"`
}

func (SSHKey) TableName() string {
	return `"pharmer"."sshKey"`
}

func encodeSSHKey(pub, priv []byte) (*SSHKey, error) {
	return nil, store.ErrNotImplemented
}

func decodeSSHKey(in *SSHKey) ([]byte, []byte, error) {
	return nil, nil, store.ErrNotImplemented
}
