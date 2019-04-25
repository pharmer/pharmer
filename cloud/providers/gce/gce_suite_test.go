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
	yaml "gopkg.in/yaml.v2"
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
