package xorm

import (
	"gomodules.xyz/secrets/types"
)

type SSHKey struct {
	ID          int64
	Name        string             `xorm:"not null 'name'"`
	ClusterID   int64              `xorm:"NOT NULL 'cluster_id'"`
	ClusterName string             `xorm:"not null 'cluster_name'"`
	UID         string             `xorm:"not null 'uid'"`
	PublicKey   string             `xorm:"text not null 'public_key'"`
	PrivateKey  types.SecureString `xorm:"text not null 'private_key'"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (SSHKey) TableName() string {
	return "ac_cluster_ssh"
}
