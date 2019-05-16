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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MachineTypeSpec defines the desired state of MachineType
type MachineTypeSpec struct {
	SKU         string             `json:"sku"`
	Description string             `json:"description,omitempty"`
	Category    string             `json:"category,omitempty"`
	CPU         *resource.Quantity `json:"cpu"`
	RAM         *resource.Quantity `json:"ram"`
	Disk        *resource.Quantity `json:"disk,omitempty"`
	Regions     []string           `json:"regions,omitempty"`
	Zones       []string           `json:"zones,omitempty"`
	Deprecated  bool               `json:"deprecated,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
// +genclient:skipVerbs=updateStatus,watch
// +kubebuilder:object:root=true

// MachineType is the Schema for the machinetypes API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="SKU",type="string",JSONPath=".spec.sku"
// +kubebuilder:printcolumn:name="CPU",type="string",JSONPath=".spec.cpu"
// +kubebuilder:printcolumn:name="RAM",type="string",JSONPath=".spec.ram"
type MachineType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MachineTypeSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
// +kubebuilder:object:root=true

// MachineTypeList contains a list of MachineType
type MachineTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineType `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MachineType{}, &MachineTypeList{})
}
