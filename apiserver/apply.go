package apiserver

import (
	"github.com/pharmer/pharmer/store"

	"github.com/golang/glog"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	opts "github.com/pharmer/pharmer/cloud/cmds/options"
)

func ApplyCluster(opt *opts.ApplyConfig, obj *api.Operation) {
	_, err := Apply(opt)
	if err != nil {
		glog.Errorf("err = %v]\n", err)
		return
	}
	obj.State = api.OperationDone
	obj, err = store.StoreProvider.Operations().Update(obj)
	if err != nil {
		glog.Errorf("[err = %v]\n", err)
	}
}
