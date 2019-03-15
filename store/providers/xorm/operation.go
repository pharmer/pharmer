package xorm

import (
	"fmt"

	"github.com/go-xorm/xorm"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
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
