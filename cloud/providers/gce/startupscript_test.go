package gce

import (
	"context"
	"fmt"
	"testing"

	"github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
)

func Test_cloudConnector_renderStartupScript(t *testing.T) {

	ctx := cloud.NewContext(context.Background(), &v1beta1.PharmerConfig{}, "")

	cluster := getCluster()
	credential := getCredential()

	_, err := cloud.Store(ctx).Credentials().Create(credential)
	if err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	_, err = cloud.Create(ctx, cluster, "")
	if err != nil {
		t.Fatalf("failed to create cluster :%v", err)
	}

	cmInterface, err := cloud.GetCloudManager("gce", ctx)
	if err != nil {
		t.Fatalf("failed to get cloud manager :%v", err)
	}

	cm, ok := cmInterface.(*ClusterManager)
	if !ok {
		t.Fatalf("failed to get cluster manager: %v", err)
	}

	cm.cluster = cluster
	cm.namer = namer{cluster: cluster}

	err = PrepareCloud(cm)
	if err != nil {
		t.Fatalf("failed to create cluster :%v", err)
	}

	machine := getMachine()

	script, err := cm.conn.renderStartupScript(cm.conn.cluster, machine, "token")
	if err != nil {
		t.Fatalf("failed to generate startupscript: %v", err)
	}

	fmt.Println(script)
}
