package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstanceGroup struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	ObjectMeta      `json:"metadata,omitempty"`
	Spec            InstanceGroupSpec   `json:"spec,omitempty"`
	Status          InstanceGroupStatus `json:"status,omitempty"`
}

type InstanceGroupSpec struct {
	SKU              string `json:"sku,omitempty"`
	Count            int64  `json:"count,omitempty"`
	UseSpotInstances bool   `json:"useSpotInstances,omitempty"`
}

type InstanceGroupStatus struct {
}

type Instance struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	ObjectMeta      `json:"metadata,omitempty"`
	Spec            InstanceSpec   `json:"spec,omitempty"`
	Status          InstanceStatus `json:"status,omitempty"`
}

type InstanceSpec struct {
	SKU  string
	Role string
}

type InstanceStatus struct {
	Name          string
	ExternalID    string
	PublicIP      string
	PrivateIP     string
	ExternalPhase string
	Phase         string
}
