package gce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"testing" //. "github.com/pharmer/pharmer/cloud/providers/gce"
	"text/template"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega" //	"time"
	cloudapi "github.com/pharmer/cloud/pkg/apis/cloud/v1"
	"github.com/pharmer/cloud/pkg/credential"
	"github.com/pharmer/pharmer/apis/v1beta1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestGce(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gce Suite")
}

func TestNG(t *testing.T) {
	cluster := "g12"
	ng := "g12-n1-standard-2"
	fmt.Println(ng[len(cluster)+1:])
}

func TestJson(t *testing.T) {
	data := ``
	crd := cloudapi.CredentialSpec{
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

func TestConfigSetup(t *testing.T) {
	tmpl, err := template.New("machine-config").Parse(machineSetupConfig)
	if err != nil {
		fmt.Println(err)
	}
	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, setupConfig{
		OS:                "ubuntu-16.04-xenial",
		OSFamily:          "ubuntu-16.04lts",
		KubernetesVersion: "v1.13.2",
	})
	fmt.Println(err)

	fmt.Println(tmplBuf.String())

	data, err := yaml.Marshal(tmplBuf.String())
	fmt.Println(string(data))

}

func TestReadfile(t *testing.T) {
	data, err := ioutil.ReadFile("machine_setup_configs.yaml")
	fmt.Println(err)
	out, err := yaml.Marshal(string(data))
	fmt.Println(err)

	fmt.Println(string(out))
}

func TestControllerManager(t *testing.T) {
}

func getCluster() *v1beta1.Cluster {
	return &v1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1beta1.PharmerClusterSpec{
			ClusterAPI: &clusterapi.Cluster{},
			Config: &v1beta1.ClusterConfig{
				Cloud: v1beta1.CloudSpec{
					NetworkProvider: v1beta1.PodNetworkCalico,
					CloudProvider:   "gce",
					Zone:            "us-central-1f",
				},
				CredentialName:    "test",
				KubernetesVersion: "v1.14.0",
			},
		},
	}
}

func getCredential() *cloudapi.Credential {
	return &cloudapi.Credential{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: cloudapi.CredentialSpec{
			Provider: "azure",
			Data: map[string]string{
				credential.GCEProjectID: "a",
				credential.GCEServiceAccount: `
{
	"type" : "service_account"
}
`,
			},
		},
	}
}

func getMachine() *clusterapi.Machine {
	return &clusterapi.Machine{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: clusterapi.MachineSpec{
			ObjectMeta:   metav1.ObjectMeta{},
			Taints:       nil,
			ProviderSpec: clusterapi.ProviderSpec{},
			Versions: clusterapi.MachineVersionInfo{
				Kubelet:      "v1.14.0",
				ControlPlane: "v1.14.0",
			},
			ConfigSource: nil,
			ProviderID:   nil,
		},
		Status: clusterapi.MachineStatus{},
	}
}
