package xorm

import (
	"gomodules.xyz/secrets/types"
)

type Certificate struct {
	ID          int64 `xorm:"pk autoincr"`
	Name        string
	ClusterID   int64 `xorm:"NOT NULL 'cluster_id'"`
	ClusterName string
	UID         string             `xorm:"uid UNIQUE"`
	Cert        types.SecureString `xorm:"text NOT NULL"`
	Key         types.SecureString `xorm:"text NOT NULL"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Certificate) TableName() string {
	return "ac_cluster_certificate"
}
