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

type Certificate struct {
	ID          int64 `xorm:"pk autoincr"`
	Name        string
	ClusterID   int64 `xorm:"NOT NULL 'cluster_id'"`
	ClusterName string
	UID         string             `xorm:"uid UNIQUE"`
	Cert        string             `xorm:"text NOT NULL"`
	Key         types.SecureString `xorm:"text NOT NULL"`

	CreatedUnix int64  `xorm:"INDEX created"`
	UpdatedUnix int64  `xorm:"INDEX updated"`
	DeletedUnix *int64 `xorm:"null"`
}

func (Certificate) TableName() string {
	return "ac_cluster_certificate"
}
