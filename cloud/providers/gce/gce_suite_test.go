package gce

import (
	//go_ctx "context"
	"fmt"
	"testing"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	_env "github.com/appscode/go/env"
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
	"regexp"

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
	ctx := NewContext(context.Background(), cfg, _env.Dev)
	cm := New(ctx)

	req := proto.ClusterCreateRequest{
		Name:     "gce-kube",
		Provider: "gce",
		Zone:     "us-central1-f",
		//CredentialUid:      "gce",
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
	/* db store instance
		dbInstances, err := Store(cm.ctx).Instances(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	existingNodes := make(map[string]*api.Node)
	for _, di := range dbInstances {
		fmt.Println(di.Name, "&&&&&&&&&&&&&&&&&&&")
		if di.Spec.Role != api.RoleMaster {
			existingNodes[di.Name] = di
			fmt.Println(di.Name, "__________ not master")
		}
	}

	fmt.Println("existing nodes = ", existingNodes)

	clusterNodes := make(map[string]*api.Node)

	for _, ng := range nodeGroups {
		fmt.Println(ng.Name)
		if ng.IsMaster() {
			continue
		}
		instances, err := cm.listInstances(cm.namer.NodeGroupName(ng.Spec.Template.Spec.SKU))
		if err != nil {
			fmt.Println(err)
			//return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		fmt.Println(instances, ".,.,.,.,.,.,.,.,.,.,")
		for _, node := range instances {
			fmt.Println("Cluster node => ", node.Name)
			if _, found := existingNodes[node.Name]; found {
				fmt.Println(node.Name, "__________ update")
				Store(cm.ctx).Instances(cm.cluster.Name).Update(node)
			} else {
				Store(cm.ctx).Instances(cm.cluster.Name).Create(node)
				fmt.Println(node.Name, "__________ create")
			}

			clusterNodes[node.Name] = node
		}
	}

	for name := range existingNodes {
		if _, found := clusterNodes[name]; !found {
			fmt.Println(name, "delete ***********************")
			Store(cm.ctx).Instances(cm.cluster.Name).Delete(name)
		}
	}

	*/

}

func TestNG(t *testing.T) {
	cluster := "g12"
	ng := "g12-n1-standard-2"
	fmt.Println(ng[len(cluster)+1:])
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

func TestRE(t *testing.T) {
	fmt.Println(TemplateURI)
	abc := regexp.MustCompile(`^` + TemplateURI + `([^/]+)/global/instanceTemplates/([^/]+)$`)
	r := abc.FindStringSubmatch("https://www.googleapis.com/compute/v1/projects/k8s-qa/global/instanceTemplates/gc1-n1-standard-2-v1508392105708944214")
	fmt.Println(len(r), r[2])
	//regexp.MustCompile(`^` + ProviderName + `://([^/]+)/([^/]+)/([^/]+)$`)
	x := providerIdRE.FindStringSubmatch("gce://k8s-qa/us-central1-f/n1-standard-2-pool-xcoq6s-rr19")
	fmt.Println(x)
}
