package fake

import (
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/store"
)

type operationFileStore struct {
	container map[string][]byte
}

var _ store.OperationStore = &operationFileStore{}

func (o *operationFileStore) Get(id string) (*api.Operation, error) {
	op := &api.Operation{
		Code: id,
	}

	return op, nil
}

func (o *operationFileStore) Update(obj *api.Operation) (*api.Operation, error) {
	return obj, nil
}
