package azure

import (
	"context"
	"fmt"
	"testing"

	"github.com/appscode/go/env"
	"github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/config"
	"github.com/pharmer/pharmer/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

var cm *ClusterManager
var conn *cloudConnector

func init() {
	conf := config.NewDefaultConfig()
	ctx := cloud.NewContext(context.Background(), conf, env.Environment(""))

	cluster := &v1beta1.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-" + util.RandomString(6),
		},
		Spec: v1beta1.PharmerClusterSpec{
			ClusterAPI: &v1alpha1.Cluster{},
			Config: &v1beta1.ClusterConfig{
				MasterCount: 0,
				Cloud: v1beta1.CloudSpec{
					CloudProvider: "azure",
					Region:        "v1beta1",
					Zone:          "v1beta1",
				},
				KubernetesVersion: "v1.14.0",

				CredentialName: "azure",
			},
		},
	}

	// setup certs
	cluster, err := cloud.Create(ctx, cluster, utils.GetLocalOwner())
	if err != nil {
		panic(err)
	}

	// create cm interface
	cmInterface := New(ctx)
	var ok bool
	cm, ok = cmInterface.(*ClusterManager)
	if !ok {
		panic(ok)
	}
	cm.SetOwner(utils.GetLocalOwner())
	cm.cluster = cluster

	// load certs
	if err = PrepareCloud(cm); err != nil {
		panic(err)
	}
}

func TestClusterManager_SetupCerts(t *testing.T) {
	err := cm.SetupCerts()
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(cloud.SSHKey(cm.ctx).PrivateKey))
	fmt.Println(string(cloud.SSHKey(cm.ctx).PublicKey))
}
