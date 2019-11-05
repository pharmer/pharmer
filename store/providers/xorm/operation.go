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
	"fmt"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"

	"github.com/go-xorm/xorm"
	"github.com/pkg/errors"
)

type operationXormStore struct {
	engine *xorm.Engine
	//owner  string
}

var _ store.OperationStore = &operationXormStore{}

func (o *operationXormStore) Get(id string) (*api.Operation, error) {
	op := &api.Operation{
		Code: id,
	}
	has, err := o.engine.Get(op)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("not found")
	}
	return op, nil
}

func (o *operationXormStore) Update(obj *api.Operation) (*api.Operation, error) {
	//op := &
	if obj == nil {
		return nil, errors.New("missing operation")
	}

	op := &api.Operation{
		UserID:    obj.UserID,
		ClusterID: obj.ClusterID,
		Code:      obj.Code,
	}
	found, err := o.engine.Get(op)
	if err != nil {
		return nil, errors.Errorf("reason %v", err)
	}
	if !found {
		return nil, errors.Errorf("operation %v not found", obj.Code)
	}
	op.State = obj.State
	_, err = o.engine.ID(op.ID).Update(op)
	return obj, err
}
