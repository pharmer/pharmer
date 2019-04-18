package gce

import (
	"context"
	"testing"

	"github.com/appscode/go/env"
	"github.com/davecgh/go-spew/spew"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/utils"
)

func Test_cloudConnector_deleteMaster(t *testing.T) {
	errch := make(chan error, 1)
	close(errch)
	for err := range errch {
		spew.Dump(err)
	}
}

var cm *ClusterManager
var conn *cloudConnector

func init() {
	conf := config.NewDefaultConfig()
	ctx := cloud.NewContext(context.Background(), conf, env.Environment(""))

	cmInterface := New(ctx)
	var ok bool
	cm, ok = cmInterface.(*ClusterManager)
	if !ok {
		panic(ok)
	}
	cm.SetOwner(utils.GetLocalOwner())

	cluster, err := cloud.Store(cm.ctx).Owner(cm.owner).Clusters().Get("gce-200")
	if err != nil {
		panic(err)
	}
	cm.cluster = cluster

	conn, err = NewConnector(cm)
	if err != nil {
		panic(err)
	}
}

func Test_cloudConnector_createLoadBalancer(t *testing.T) {
	ip, err := conn.createLoadBalancer(conn.namer.loadBalancerName())
	if err != nil {
		t.Error(err)
	}

	if ip == "" {
		t.Errorf("load balancer ip address can't be empty")
	}
}

func Test_cloudConnector_deleteLoadBalancer(t *testing.T) {
	err := conn.deleteLoadBalancer()
	if err != nil {
		t.Error(err)
	}
}
