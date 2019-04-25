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

// CredentialFormatSpec defines the desired state of CredentialFormat
type CredentialFormatSpec struct {
	Provider      string            `json:"provider"`
	DisplayFormat string            `json:"displayFormat"`
	Fields        []CredentialField `json:"fields"`
}

type CredentialField struct {
	Envconfig string `json:"envconfig,omitempty"`
	Form      string `json:"form,omitempty"`
	JSON      string `json:"json,omitempty"`
	Label     string `json:"label,omitempty"`
	Input     string `json:"input,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
// +genclient:skipVerbs=updateStatus,watch

// CredentialFormat is the Schema for the credentialformats API
// +k8s:openapi-gen=true
type CredentialFormat struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CredentialFormatSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// CredentialFormatList contains a list of CredentialFormat
type CredentialFormatList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CredentialFormat `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CredentialFormat{}, &CredentialFormatList{})
}
