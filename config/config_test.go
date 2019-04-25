package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//m1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfig(t *testing.T) {
	cred := api.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "gce",
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.CredentialSpec{
			Provider: "GoogleCloud",
			Data:     make(map[string]string),
		},
	}
	cred.Spec.Data["projectID"] = ""
	cred.Spec.Data["serviceAccount"] = ``
	conf := &api.PharmerConfig{
		Context: "default",
		Credentials: []api.Credential{
			cred,
		},
		Store: api.StorageBackend{
			CredentialName: "gce",
			GCS: &api.GCSSpec{
				Bucket: "pharmer",
				Prefix: "",
			},
		},
	}
	data, err := yaml.Marshal(conf)
	fmt.Println(string(data), err)
}
