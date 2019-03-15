package xorm

import (
	"fmt"
	"testing"
	"time"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pharmer/pharmer/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getEngine() store.Interface {
	engine, err := newPGEngine("postgres", "postgres", "127.0.0.1", 5432, "postgres")
	fmt.Println(err)
	return New(engine)
}

func TestCredentialCreate(t *testing.T) {
	//x := getEngine()
	cred := &api.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "do",
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Spec: api.CredentialSpec{
			Provider: "digitalocean",
			Data:     make(map[string]string),
		},
	}
	data := map[string]string{
		"token": "1111111111111111",
	}
	cred.Spec.Data = data
	//_, err := x.Credentials().Create(cred)
	//fmt.Println(err)

}

func TestCredentialGet(t *testing.T) {
	x := getEngine()
	cred, err := x.Credentials().Get("do")
	fmt.Println(cred, err)
}
