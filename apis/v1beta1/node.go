package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NodeGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NodeGroupSpec   `json:"spec,omitempty"`
	Status            NodeGroupStatus `json:"status,omitempty"`
}

type NodeGroupSpec struct {
	Nodes int64 `json:"nodes"`
	// Template describes the nodes that will be created.
	Template NodeTemplateSpec `json:"template"`
}

// NodeGroupStatus is the most recently observed status of the NodeGroup.
type NodeGroupStatus struct {
	// Nodes is the most recently oberved number of nodes.
	Nodes int64 `json:"nodes"`
	// ObservedGeneration reflects the generation of the most recently observed node group.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

func (ng NodeGroup) IsMaster() bool {
	_, found := ng.Labels[RoleMasterKey]
	return found
}

func (ng NodeGroup) Role() string {
	if ng.IsMaster() {
		return RoleMaster
	}
	return RoleNode
}

// PodTemplateSpec describes the data a pod should have when created from a template
type NodeTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	// metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec NodeSpec `json:"spec,omitempty"`
}
