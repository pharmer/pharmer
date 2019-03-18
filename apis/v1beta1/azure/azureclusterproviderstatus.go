package azure

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureClusterProviderStatus contains the status fields
// relevant to Azure in the cluster object.
// +k8s:openapi-gen=true
type AzureClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Network Network `json:"network,omitempty"`
	Bastion VM      `json:"bastion,omitempty"`
}
