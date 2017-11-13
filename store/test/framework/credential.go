package framework

import (
	"fmt"
	"time"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *credentialInvocation) GetName() string {
	return "test-do"
}
func (c *credentialInvocation) GetSkeleton() *api.Credential {
	cred := &api.Credential{
		ObjectMeta: metav1.ObjectMeta{
			Name:              c.GetName(),
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
	return cred
}

func (c *credentialInvocation) Update(cred *api.Credential) error {
	data := map[string]string{
		"token": "22222222222222222",
	}
	cred.Spec.Data = data
	_, err := c.Storage.Credentials().Update(cred)
	return err
}

func (c *credentialInvocation) CheckUpdate(cred *api.Credential) error {
	data := cred.Spec.Data
	if token, ok := data["token"]; ok {
		if token == "22222222222222222" {
			return nil
		}
	}
	return fmt.Errorf("credential was not updated")
}

func (c *credentialInvocation) List() error {
	clusters, err := c.Storage.Credentials().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(clusters) < 1 {
		return fmt.Errorf("can't list crdentials")
	}
	return nil
}
