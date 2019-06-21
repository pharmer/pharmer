package apiserver

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
)

func ApplyCluster(scope *cloud.Scope, obj *api.Operation) error {
	scope.Logger.Info("applying cluster")

	err := cloud.Apply(scope)
	if err != nil {
		return err
	}
	obj.State = api.OperationDone
	_, err = scope.StoreProvider.Operations().Update(obj)
	if err != nil {
		return err
	}

	scope.Logger.Info("successfully applied cluster")

	return nil
}
