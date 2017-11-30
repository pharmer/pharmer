package gce

import (
	//go_ctx "context"
	"fmt"
	"testing"
	//. "github.com/pharmer/pharmer/cloud/providers/gce"
	//"github.com/pharmer/pharmer/config"
	//"github.com/pharmer/pharmer/context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//	"time"
	//	api "github.com/pharmer/pharmer/apis/v1alpha1"
	//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"encoding/json"
	"regexp"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
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
