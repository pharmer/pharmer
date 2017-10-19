package v1alpha1

import (
	"github.com/appscode/mergo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

var _ runtime.Object = &Action{}

func (n *Action) DeepCopyObject() runtime.Object {
	if n == nil {
		return n
	}
	out := new(Action)
	mergo.MergeWithOverwrite(out, n)
	return out
}
