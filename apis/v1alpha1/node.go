package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceCodeNodeGroup = "ng"
	ResourceKindNodeGroup = "NodeGroup"
	ResourceNameNodeGroup = "nodegroup"
	ResourceTypeNodeGroup = "nodegroups"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NodeGroup struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              NodeGroupSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            NodeGroupStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
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
	// metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec NodeSpec `json:"spec,omitempty"`
}

type IPType string

const (
	IPTypeEphemeral IPType = "Ephemeral"
	IPTypeReserved  IPType = "Reserved"
)

type NodeSpec struct {
	SKU              string            `json:"sku,omitempty"`
	SpotInstances    bool              `json:"spotInstances,omitempty"`
	DiskType         string            `json:"nodeDiskType,omitempty"`
	DiskSize         int64             `json:"nodeDiskSize,omitempty"`
	ExternalIPType   IPType            `json:"externalIPType,omitempty"`
	KubeletExtraArgs map[string]string `json:"kubeletExtraArgs,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type NodeInfo struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	Name            string `json:"name,omitempty"`
	ExternalID      string `json:"externalID,omitempty"`
	PublicIP        string `json:"publicIP,omitempty"`
	PrivateIP       string `json:"privateIP,omitempty"`
	DiskId          string `json:"diskID,omitempty"`
}
