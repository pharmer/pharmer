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
package v1alpha1

type OperationState int

const (
	OperationDone OperationState = iota // 0
	OperationPending
	OperationRunning
)

type Operation struct {
	ID        int64  `xorm:"pk autoincr 'id'"`
	UserID    int64  `xorm:"UNIQUE(s) 'user_id'"`
	ClusterID int64  `xorm:"UNIQUE(s) 'cluster_id'"`
	Code      string `xorm:"UNIQUE(s)"`
	State     OperationState
}

func (Operation) TableName() string {
	return "ac_operation"
}
