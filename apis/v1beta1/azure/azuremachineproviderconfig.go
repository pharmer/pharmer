package azure

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureMachineProviderSpec is the Schema for the azuremachineproviderspecs API
// +k8s:openapi-gen=true
type AzureMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Roles         []MachineRole `json:"roles,omitempty"`
	Location      string        `json:"location"`
	VMSize        string        `json:"vmSize"`
	Image         Image         `json:"image"`
	OSDisk        OSDisk        `json:"osDisk"`
	SSHPublicKey  string        `json:"sshPublicKey"`
	SSHPrivateKey string        `json:"sshPrivateKey"`
}
