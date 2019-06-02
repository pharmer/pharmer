package cmds

import (
	"encoding/json"
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"time"

	. "github.com/onsi/ginkgo"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("cluste", func() {

		cluster := &api.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "testdo",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: api.PharmerClusterSpec{
				ClusterAPI: &clusterapi.Cluster{},
				Config: &api.ClusterConfig{
					CredentialName:    "docred",
					KubernetesVersion: "v1.13.0",
					Cloud: api.CloudSpec{
						CloudProvider:   "digitalocean",
						Zone:            "nyc3",
						NetworkProvider: api.PodNetworkCalico,
					},
				},
			},
		}
		d, e := json.Marshal(cluster)
		fmt.Println(e)
		fmt.Println(string(d))
	})
})
