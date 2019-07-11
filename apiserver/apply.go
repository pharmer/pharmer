package apiserver

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
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
