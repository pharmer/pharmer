package xorm

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
)

func TestJM(t *testing.T) {
	fmt.Println(os.Getenv("HOME"))
}

func TestClusterEngine(t *testing.T) {
	////engine, err := newPGEngine("postgres", "postgres", "127.0.0.1", 5432, "postgres")
	////fmt.Println(err)
	////x := New(engine)
	//cluster := &api.Cluster{}
	//cluster.Name = "xorm-test"
	//cluster.Spec.Config.Cloud.CloudProvider = "digitalocean"
	//cluster.Spec.Config.Cloud.Zone = "nyc3"
	//cluster.Spec.Config.CredentialName = "do"
	//cluster.Spec.Config.KubernetesVersion = "v1.9.0"
	//
	//cluster.Spec.Config.Networking.NetworkProvider = "calico"
	//// Init object meta
	//cluster.ObjectMeta.UID = uuid.NewUUID()
	//cluster.ObjectMeta.CreationTimestamp = metav1.Time{Time: time.Now()}
	//cluster.ObjectMeta.Generation = time.Now().UnixNano()
	//api.AssignTypeKind(cluster)
	//
	//// Init spec
	//cluster.Spec.Config.Cloud.Region = cluster.Spec.Config.Cloud.Zone
	////cluster.Spec.API.BindPort = kubeadmapi.DefaultAPIBindPort
	//cluster.Spec.Config.Cloud.InstanceImage = "ubuntu-16-04-x64"
	////cluster.Spec.AuthorizationModes = strings.Split(kubeadmapi.DefaultAuthorizationModes, ",")
	////cluster.Spec.APIServerCertSANs = ""
	//cluster.Spec.Config.APIServerExtraArgs = map[string]string{
	//	// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
	//	"kubelet-preferred-address-types": strings.Join([]string{
	//		string(core.NodeInternalIP),
	//		string(core.NodeExternalIP),
	//	}, ","),
	//}
	//// Init status
	//cluster.Status = api.ClusterStatus{
	//	Phase: api.ClusterPending,
	//}
	//
	////_, err = x.Clusters().Create(cluster)
	////fmt.Println(err)
}

func TestCred(t *testing.T) {
	data := `{"kind":"Credential","apiVersion":"v1alpha1","metadata":{"name":"do2","creationTimestamp":"2018-11-21T06:54:35Z"},"spec":{"provider":"DigitalOcean","data":{"token":"testcredential"}}}`

	obj := cloudapi.Credential{}
	err := json.Unmarshal([]byte(data), &obj)
	fmt.Println(err)
	fmt.Println(obj.Kind)
}
