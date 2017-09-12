package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

type NodeGroup struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NodeGroupSpec   `json:"spec,omitempty"`
	Status            NodeGroupStatus `json:"status,omitempty"`
}

type NodeGroupSpec struct {
	Nodes int64 `json:"nodes,omitempty"`

	// Template describes the nodes that will be created.
	Template NodeTemplateSpec `json:"template" protobuf:"bytes,3,opt,name=template"`
}

// NodeGroupStatus is the most recently observed status of the NodeGroup.
type NodeGroupStatus struct {
	// Nodes is the most recently oberved number of nodes.
	Nodes int32 `json:"nodes" protobuf:"varint,1,opt,name=nodes"`

	// The number of pods that have labels matching the labels of the pod template of the node set.
	// +optional
	FullyLabeledNodes int32 `json:"fullyLabeledNodes,omitempty" protobuf:"varint,2,opt,name=fullyLabeledNodes"`

	// The number of ready nodes for this node set.
	// +optional
	ReadyNodes int32 `json:"readyNodes,omitempty" protobuf:"varint,4,opt,name=readyNodes"`

	// The number of available nodes (ready for at least minReadySeconds) for this node set.
	// +optional
	AvailableNodes int32 `json:"availableNodes,omitempty" protobuf:"varint,5,opt,name=availableNodes"`

	// ObservedGeneration reflects the generation of the most recently observed ReplicaSet.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`

	// Represents the latest available observations of a node set's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []NodeGroupCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,6,rep,name=conditions"`
}

type NodeGroupConditionType string

// These are valid conditions of a deployment.
const (
	// Available means the deployment is available, ie. at least the minimum available
	// nodes required are up and running for at least minReadySeconds.
	NodeGroupAvailable NodeGroupConditionType = "Available"
	// Progressing means the deployment is progressing. Progress for a deployment is
	// considered when a new node set is created or adopted, and when new nodes scale
	// up or old nodes scale down. Progress is not estimated for paused deployments or
	// when progressDeadlineSeconds is not specified.
	NodeGroupProgressing NodeGroupConditionType = "Progressing"
	// ReplicaFailure is added in a deployment when one of its nodes fails to be created
	// or deleted.
	NodeGroupReplicaFailure NodeGroupConditionType = "ReplicaFailure"
)

// NodeGroupCondition describes the state of a deployment at a certain point.
type NodeGroupCondition struct {
	// Type of deployment condition.
	Type NodeGroupConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=NodeGroupConditionType"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" protobuf:"bytes,6,opt,name=lastUpdateTime"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,7,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

func (ns NodeGroup) IsMaster() bool {
	_, found := ns.Labels[RoleMasterKey]
	return found
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
	Spec NodeSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// Deprecated, replace with Kubernetes Node
type Node struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NodeSpec   `json:"spec,omitempty"`
	Status            NodeStatus `json:"status,omitempty"`
}

// Deprecated
type NodeSpec struct {
	// Deprecated
	Role string

	SKU           string `json:"sku,omitempty"`
	SpotInstances bool   `json:"spotInstances,omitempty"`
	DiskType      string `json:"nodeDiskType,omitempty"`
	DiskSize      int64  `json:"nodeDiskSize,omitempty"`
}

// Deprecated
type NodeStatus struct {
	Phase NodePhase

	Name          string
	ExternalID    string
	PublicIP      string
	PrivateIP     string
	ExternalPhase string
	DiskId        string `json:"diskID,omitempty"`
}

func (n Node) IsMaster() bool {
	_, found := n.Labels[RoleMasterKey]
	return found
}

// InstancePhase is a label for the condition of an Instance at the current time.
// Deprecated
type NodePhase string

const (
	NodeReady   NodePhase = "Ready"
	NodeDeleted NodePhase = "Deleted"
)
