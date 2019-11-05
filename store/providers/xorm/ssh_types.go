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
