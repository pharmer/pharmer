package framework

import (
	"time"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *credentialInvocation) GetName() string {
	return "do"
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
