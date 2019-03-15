package aws

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSClusterProviderStatus contains the status fields
// relevant to AWS in the cluster object.
// +k8s:openapi-gen=true
type AWSClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Network Network  `json:"network,omitempty"`
	Bastion Instance `json:"bastion,omitempty"`
}
