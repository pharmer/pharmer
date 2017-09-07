package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstanceGroup struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstanceGroupSpec   `json:"spec,omitempty"`
	Status            InstanceGroupStatus `json:"status,omitempty"`
}

type InstanceGroupSpec struct {
	SKU           string `json:"sku,omitempty"`
	Count         int64  `json:"count,omitempty"`
	SpotInstances bool   `json:"spotInstances,omitempty"`
	DiskType      string `json:"nodeDiskType,omitempty"`
	DiskSize      int64  `json:"nodeDiskSize,omitempty"`
}

type InstanceGroupStatus struct {
}

func (ig InstanceGroup) IsMaster() bool {
	_, found := ig.Labels[RoleMasterKey]
	return found
}

type Instance struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstanceSpec   `json:"spec,omitempty"`
	Status            InstanceStatus `json:"status,omitempty"`
}

type InstanceSpec struct {
	// Deprecated
	Role string

	SKU           string `json:"sku,omitempty"`
	SpotInstances bool   `json:"spotInstances,omitempty"`
	DiskType      string `json:"nodeDiskType,omitempty"`
	DiskSize      int64  `json:"nodeDiskSize,omitempty"`
}

type InstanceStatus struct {
	Phase InstancePhase

	Name          string
	ExternalID    string
	PublicIP      string
	PrivateIP     string
	ExternalPhase string
	DiskId        string `json:"diskID,omitempty"`
}

func (i Instance) IsMaster() bool {
	_, found := i.Labels[RoleMasterKey]
	return found
}

// InstancePhase is a label for the condition of an Instance at the current time.
type InstancePhase string

const (
	InstanceReady   InstancePhase = "Ready"
	InstanceDeleted InstancePhase = "Deleted"
)
