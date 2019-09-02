package xorm

import (
	"gomodules.xyz/secrets/types"
)

type Machine struct {
	ID        int64              `xorm:"pk autoincr"`
	Name      string             `xorm:"INDEX NOT NULL"`
	Data      types.SecureString `xorm:"text NOT NULL"`
	ClusterID int64              `xorm:"INDEX NOT NULL"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Machine) TableName() string {
	return "ac_cluster_machine"
}
