package gce

import (
	"bytes"
	"io/ioutil"
	"text/template"

	"gopkg.in/yaml.v2"

	//go_ctx "context"
	"encoding/json"
	"fmt"
	"regexp"
	"testing" //. "github.com/pharmer/pharmer/cloud/providers/gce"

	//"github.com/pharmer/pharmer/config"
	//"github.com/pharmer/pharmer/context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega" //	"time"

	//	api "github.com/pharmer/pharmer/apis/v1alpha1"
	//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	apibeta "github.com/pharmer/pharmer/apis/v1beta1"
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

func TestConfigSetup(t *testing.T) {
	tmpl, err := template.New("machine-config").Parse(machineSetupConfig)
	if err != nil {
		fmt.Println(err)
	}
	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, setupConfig{
		OS:             "ubuntu-16.04-xenial",
		OSFamily:       "ubuntu-16.04lts",
		KubeletVersion: "v1.13.2",

		ControlPlaneVersion: "v1.13.2",
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
	tmpl, err := template.New("controller-manager-config").Parse(ControllerManager)
	fmt.Println(err)
	var tmplBuf bytes.Buffer

	machinsetup, err := geteMachineSetupConfig(&apibeta.ClusterConfig{
		Cloud: apibeta.CloudSpec{
			InstanceImage: "ubuntu-16.04-xenial",
			OS:            "ubuntu",
		},
		KubernetesVersion: "v1.13.2",
	})
	fmt.Println(machinsetup, err)

	err = tmpl.Execute(&tmplBuf, controllerManagerConfig{
		MachineConfig:  machinsetup,
		ServiceAccount: "abc",
		SSHPrivateKey:  "pqr",
		SSHPublicKey:   "xyz",
		SSHUser:        "clusterapi",
	})
	fmt.Println(err)

	fmt.Println(tmplBuf.String(), err)
}
