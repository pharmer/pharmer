package aws

import (
	"context"
	"fmt"
	"testing"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAws(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aws Suite")
}

func TestContext(t *testing.T) {
	cfg, err := config.LoadConfig("/home/sanjid/go/src/appscode.com/ark/conf/aws.json")
	fmt.Println(err)
	ctx := cloud.NewContext(context.TODO(), cfg)
	cm := New(ctx)

	req := proto.ClusterCreateRequest{
		Name:               "gce-kube",
		Provider:           "gce",
		Zone:               "us-east-1",
		CredentialUid:      "aws",
		DoNotDelete:        false,
		DefaultAccessLevel: "kubernetes:cluster-admin",
	}
	req.NodeGroups = make([]*proto.InstanceGroup, 1)
	req.NodeGroups[0] = &proto.InstanceGroup{
		Sku:   "n1-standard-1",
		Count: int64(1),
	}
	cm, err = cloud.GetCloudManager(req.Provider, ctx)
	fmt.Println(err, cm)

	// cm.Check(&req)

	/*cm.cluster = &api.Cluster{
		ObjectMeta: api.ObjectMeta{
			Name:              req.Name,
			UID:               phid.NewKubeCluster(),
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.ClusterSpec{
			CredentialName: req.CredentialUid,
		},
	}
	cm.cluster.Spec.Zone = req.Zone

	api.AssignTypeKind(cm.cluster)
	if _, err := cloud.Store(cm.ctx).Clusters().Create(cm.cluster); err != nil {
		//oneliners.FILE(err)
		cm.cluster.Status.Reason = err.Error()
		fmt.Println(err)
	}

	err := cm.initContext(&req)
	fmt.Println(err)
	fmt.Println(cm.ctx)
	*/ /*cm.Check(&proto.ClusterCreateRequest{
		Name:     "test",
		Provider: "gce",
		Zone:     "us-central1-f",
	})*/ /*
		fmt.Println()*/

}
