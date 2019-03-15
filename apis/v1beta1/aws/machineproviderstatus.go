package aws

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSMachineProviderStatus is the type that will be embedded in a Machine.Status.ProviderStatus field.
// It containsk AWS-specific status information.
// +k8s:openapi-gen=true
type AWSMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// InstanceID is the instance ID of the machine created in AWS
	// +optional
	InstanceID *string `json:"instanceID,omitempty"`

	// InstanceState is the state of the AWS instance for this machine
	// +optional
	InstanceState *InstanceState `json:"instanceState,omitempty"`

	// Conditions is a set of conditions associated with the Machine to indicate
	// errors or other status
	// +optional
	Conditions []AWSMachineProviderCondition `json:"conditions,omitempty"`
}
