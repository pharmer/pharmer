package azure

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureMachineProviderStatus is the type that will be embedded in a Machine.Status.ProviderStatus field.
// It contains Azure-specific status information.
// +k8s:openapi-gen=true
type AzureMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VMID is the instance ID of the machine created in Azure.
	// +optional
	VMID *string `json:"vmId,omitempty"`

	// VMState is the state of the Azure instance for this machine.
	// +optional
	VMState *string `json:"instanceState,omitempty"`

	// Conditions is a set of conditions associated with the Machine to indicate
	// errors or other status.
	// +optional
	Conditions []AzureMachineProviderCondition `json:"conditions,omitempty"`
}
