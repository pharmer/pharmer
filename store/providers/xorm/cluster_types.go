package xorm

import (
	"gomodules.xyz/secrets/types"
)

// Cluster represents a kubernets cluster
type Cluster struct {
	ID        int64              `xorm:"pk autoincr"`
	UUID      string             `xorm:"uuid UNIQUE"`
	OwnerID   int64              `xorm:"UNIQUE(s)"`
	Name      string             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Data      types.SecureString `xorm:"text NOT NULL"`
	IsPrivate bool               `xorm:"INDEX"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Cluster) TableName() string {
	return "ac_cluster"
}
