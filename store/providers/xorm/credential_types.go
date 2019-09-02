package xorm

import (
	"gomodules.xyz/secrets/types"
)

type Credential struct {
	ID      int64              `xorm:"pk autoincr"`
	OwnerID int64              `xorm:"UNIQUE(s)"`
	Name    string             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	UID     string             `xorm:"uid UNIQUE"`
	Data    types.SecureString `xorm:"text NOT NULL"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Credential) TableName() string {
	return "ac_cluster_credential"
}
