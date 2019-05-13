package vultr

import (
	. "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	VultrProviderGroupName  = "vultrproviderconfig"
	VultrProviderKind       = "VultrClusterProviderConfig"
	VultrProviderApiVersion = "v1alpha1"
)

type VultrClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type VultrClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type VultrMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Roles  []MachineRole `json:"roles,omitempty"`
	Region string        `json:"region,omitempty"`
	Plan   string        `json:"plan,omitempty"`
	Image  string        `json:"image,omitempty"`
	Pubkey string        `json:"pubkey,omitempty"`
}

type VultrMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	InstanceID     string `json:"instanceID"`
	InstanceStatus string `json:"instanceStatus"`
}
