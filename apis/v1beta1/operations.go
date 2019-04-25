package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ActionType string

const (
	ActionNOP    ActionType = "Nop"
	ActionAdd    ActionType = "Add"
	ActionUpdate ActionType = "Update"
	ActionDelete ActionType = "Delete"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Action struct {
	metav1.TypeMeta `json:",inline"`
	Action          ActionType `json:"action,omitempty"`
	Resource        string     `json:"resource,omitempty"`
	Message         string     `json:"message,omitempty"`
}
