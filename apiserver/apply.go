package apiserver

import (
	"context"
	opts "github.com/pharmer/pharmer/cloud/cmds/options"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/golang/glog"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

func ApplyCluster(ctx context.Context, opt *opts.ApplyConfig, obj *api.Operation)  {
	_, err := Apply(ctx, opt)
	if err != nil {
		glog.Errorf("err = %v]\n", err)
	}
	obj.State = api.OperationDone
	obj, err = Store(ctx).Owner(opt.Owner).Operations().Update(obj)
	if err != nil {
		glog.Errorf("[err = %v]\n", err)
	}
}
