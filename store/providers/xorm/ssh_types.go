package xorm

import (
	"time"
)

type SSHKey struct {
	Id                int64
	Name              string     `xorm:"text not null 'name'"`
	ClusterName       string     `xorm:"text not null 'clusterName'"`
	UID               string     `xorm:"text not null 'uid'"`
	PublicKey         string     `xorm:"string  not null 'publicKey'"`
	PrivateKey        string     `xorm:"string  not null 'privateKey'"`
	CreationTimestamp time.Time  `xorm:"bigint created 'creationTimestamp'"`
	DateModified      time.Time  `xorm:"bigint updated 'dateModified'"`
	DeletionTimestamp *time.Time `xorm:"bigint null 'deletionTimestamp'"`
	ClusterId         int64      `xorm:"bigint not null 'clusterId'"`
}

func (SSHKey) TableName() string {
	return `"cluster_sshKey"`
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
