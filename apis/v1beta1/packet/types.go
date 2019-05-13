package packet

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PacketProviderGroupName  = "Packetproviderconfig"
	PacketProviderKind       = "PacketClusterProviderConfig"
	PacketProviderApiVersion = "v1alpha1"
)

type PacketClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type PacketClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type PacketMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Plan         string `json:"region,omitempty"`
	SpotInstance string `json:"type,omitempty"`
}

type PacketMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	InstanceID string `json:"instanceID,omitempty"`
}
