package apiserver

import (
	"github.com/golang/glog"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	opts "github.com/pharmer/pharmer/cmds/cloud/options"
	"github.com/pharmer/pharmer/store"
)

func ApplyCluster(storeProvider store.Interface, opt *opts.ApplyConfig, obj *api.Operation) {
	err := Apply(opt, storeProvider.Owner(opt.Owner))
	if err != nil {
		glog.Errorf("err = %v]\n", err)
		return
	}
	obj.State = api.OperationDone
	_, err = storeProvider.Operations().Update(obj)
	if err != nil {
		glog.Errorf("[err = %v]\n", err)
	}
}
