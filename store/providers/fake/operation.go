package fake

import (
	"fmt"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
)

type operationFileStore struct {
	container map[string][]byte
	mux       sync.Mutex
}

var _ store.OperationStore = &operationFileStore{}

func (o *operationFileStore) Get(id string) (*api.Operation, error) {
	fmt.Println("LLLLLLLLLL")
	op := &api.Operation{
		Code: id,
	}

	return op, nil
}

func (o *operationFileStore) Update(obj *api.Operation) (*api.Operation, error) {
	return obj, nil
}
