package apiserver

import (
	"github.com/golang/glog"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	opts "github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
)

func ApplyCluster(opt *opts.ApplyConfig, obj *api.Operation) {
	err := cloud.Apply(opt)
	if err != nil {
		glog.Errorf("err = %v]\n", err)
		return
	}
	obj.State = api.OperationDone
	_, err = store.StoreProvider.Operations().Update(obj)
	if err != nil {
		glog.Errorf("[err = %v]\n", err)
	}
}
