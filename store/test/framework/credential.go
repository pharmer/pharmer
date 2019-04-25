package framework

import (
	"time"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
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
	return errors.Errorf("credential was not updated")
}

func (c *credentialInvocation) List() error {
	clusters, err := c.Storage.Credentials().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(clusters) < 1 {
		return errors.Errorf("can't list crdentials")
	}
	return nil
}
