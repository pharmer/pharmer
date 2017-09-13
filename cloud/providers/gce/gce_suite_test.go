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
	. "github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
)

func TestGce(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gce Suite")
}

func TestContext(t *testing.T) {
	cfg, err := config.LoadConfig("/home/sanjid/go/src/appscode.com/ark/conf/tigerworks-kube.json")
	fmt.Println(err)
	ctx := NewContext(context.Background(), cfg)
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
	/*req.NodeGroups = make([]*proto.NodeGroup, 1)
	req.NodeGroups[0] = &proto.NodeGroup{
		Sku:   "n1-standard-1",
		Count: int64(1),
	}*/
	cm, err = GetCloudManager(req.Provider, ctx)
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
	if _, err := Store(cm.ctx).Clusters().Create(cm.cluster); err != nil {
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

func TestJson(t *testing.T) {
	data := ``
	crd := api.CredentialSpec{
		Data: map[string]string{
			"projectID":      "tigerworks-kube",
			"serviceAccount": data,
		},
	}
	jsn, err := json.Marshal(crd)
	fmt.Println(string(jsn), err)
}

func TestNG(t *testing.T) {
	cluster := "g12"
	ng := "g12-n1-standard-2"
	fmt.Println(ng[len(cluster)+1:])
}
