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

// KubernetesVersionSpec defines the desired state of KubernetesVersion
type KubernetesVersionSpec struct {
	Major      string          `json:"major,omitempty"`
	Minor      string          `json:"minor,omitempty"`
	GitVersion string          `json:"gitVersion"`
	Envs       map[string]bool `json:"envs,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
// +genclient:skipVerbs=updateStatus,watch

// KubernetesVersion is the Schema for the kubernetesversions API
// +k8s:openapi-gen=true
type KubernetesVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KubernetesVersionSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// KubernetesVersionList contains a list of KubernetesVersion
type KubernetesVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubernetesVersion{}, &KubernetesVersionList{})
}
