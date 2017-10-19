package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceCodeCredential = "cred"
	ResourceKindCredential = "Credential"
	ResourceNameCredential = "credential"
	ResourceTypeCredential = "credentials"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Credential struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CredentialSpec `json:"spec,omitempty"`
}

type CredentialSpec struct {
	Provider string            `json:"provider"`
	Data     map[string]string `json:"data"`
}
