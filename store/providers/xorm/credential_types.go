/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
