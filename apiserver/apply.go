package apiserver

import (
	"context"

	"github.com/golang/glog"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	opts "github.com/pharmer/pharmer/cloud/cmds/options"
)

func ApplyCluster(ctx context.Context, opt *opts.ApplyConfig, obj *api.Operation) {
	_, err := Apply(ctx, opt)
	if err != nil {
		glog.Errorf("err = %v]\n", err)
		return
	}
	obj.State = api.OperationDone
	obj, err = Store(ctx).Owner(opt.Owner).Operations().Update(obj)
	if err != nil {
		glog.Errorf("[err = %v]\n", err)
	}
}
