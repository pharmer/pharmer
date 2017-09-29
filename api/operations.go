package api

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

type Action struct {
	metav1.TypeMeta `json:",inline,omitempty,omitempty"`

	Action   ActionType
	Resource string
	Message  string
}
