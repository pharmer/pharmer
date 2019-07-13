package apiserver

import (
	"github.com/pkg/errors"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cloud"
)

func ApplyCluster(scope *cloud.Scope, obj *api.Operation) error {
	err := cloud.Apply(scope)
	if err != nil {
		return err
	}
	obj.State = api.OperationDone
	_, err = scope.StoreProvider.Operations().Update(obj)
	if err != nil {
		return errors.Wrap(err, "failed to update operations in store")
	}

	return nil
}
