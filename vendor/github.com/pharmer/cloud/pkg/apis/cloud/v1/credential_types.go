/*
Copyright 2019 The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceCodeCredential = "cred"
	ResourceKindCredential = "Credential"
	ResourceNameCredential = "credential"
	ResourceTypeCredential = "credentials"

	ResourceProviderCredential = "provider"
)

// CredentialSpec defines the desired state of Credential
type CredentialSpec struct {
	Provider string            `json:"provider"`
	Data     map[string]string `json:"data"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// Credential is the Schema for the credentials API
// +k8s:openapi-gen=true
type Credential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CredentialSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// CredentialList contains a list of Credential
type CredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Credential `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Credential{}, &CredentialList{})
}
