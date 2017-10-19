package v1alpha1

import (
	"github.com/appscode/mergo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ResourceCodeCredential = "cred"
	ResourceKindCredential = "Credential"
	ResourceNameCredential = "credential"
	ResourceTypeCredential = "credentials"
)

type Credential struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CredentialSpec `json:"spec,omitempty"`
}

type CredentialSpec struct {
	Provider string            `json:"provider"`
	Data     map[string]string `json:"data"`
}

var _ runtime.Object = &Credential{}

func (c *Credential) DeepCopyObject() runtime.Object {
	if c == nil {
		return c
	}
	out := new(Credential)
	mergo.MergeWithOverwrite(out, c)
	return out
}
