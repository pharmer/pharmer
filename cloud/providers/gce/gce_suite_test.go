package gce

import (
	//go_ctx "context"
	"fmt"
	"testing"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	//. "github.com/appscode/pharmer/cloud/providers/gce"
	//"github.com/appscode/pharmer/config"
	//"github.com/appscode/pharmer/context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//	"time"
	//	"github.com/appscode/pharmer/api"
	//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"github.com/appscode/pharmer/phid"
	"context"
	"encoding/json"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
)

func TestGce(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gce Suite")
}

func TestContext(t *testing.T) {
	cfg, err := config.LoadConfig("/home/sanjid/go/src/appscode.com/ark/conf/tigerworks-kube.json")
	fmt.Println(err)
	ctx := cloud.NewContext(context.Background(), cfg)
	cm := New(ctx)

	req := proto.ClusterCreateRequest{
		Name:               "gce-kube",
		Provider:           "gce",
		Zone:               "us-central1-f",
		CredentialUid:      "gce",
		DoNotDelete:        false,
		DefaultAccessLevel: "kubernetes:cluster-admin",
		GceProject:         "tigerworks-kube",
	}
	/*req.NodeSets = make([]*proto.NodeSet, 1)
	req.NodeSets[0] = &proto.NodeSet{
		Sku:   "n1-standard-1",
		Count: int64(1),
	}*/
	cm, err = cloud.GetCloudManager(req.Provider, ctx)
	fmt.Println(err, cm)

	/*cm.cluster = &api.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              req.Name,
			UID:               phid.NewKubeCluster(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.ClusterSpec{
			CredentialName: req.CredentialUid,
		},
	}
	cm.cluster.Spec.Cloud.Zone = req.Zone

	api.AssignTypeKind(cm.cluster)
	if _, err := cloud.Store(cm.ctx).Clusters().Create(cm.cluster); err != nil {
		//oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		fmt.Println(err)
	}

	err := cm.DefaultSpec(&req)
	fmt.Println(err)
	fmt.Println(cm.ctx)
	*/ /*cm.Check(&proto.ClusterCreateRequest{
		Name:     "test",
		Provider: "gce",
		Zone:     "us-central1-f",
	})*/ /*
		fmt.Println()*/

}
