package linode

import (
	"github.com/linode/linodego"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "pharmer.dev/pharmer/apis/v1beta1"
)

const (
	LinodeProviderGroupName  = "linodeproviderconfig"
	LinodeProviderKind       = "LinodeClusterProviderConfig"
	LinodeProviderAPIVersion = "v1alpha1"
)

type LinodeMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Roles  []api.MachineRole `json:"roles,omitempty"`
	Region string            `json:"region,omitempty"`
	Type   string            `json:"type,omitempty"`
	Image  string            `json:"image,omitempty"`
	Pubkey string            `json:"pubkey,omitempty"`
}

type LinodeMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	InstanceID     int    `json:"instanceID"`
	InstanceStatus string `json:"instanceStatus"`
}

type LinodeClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Network Network `json:"network,omitempty"`
}

type LinodeClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// Network encapsulates AWS networking resources.
type Network struct {
	APIServerLB *LinodeNodeBalancer `json:"apiServerLb,omitempty"`
}

// NodeBalancer represents a NodeBalancer object
type LinodeNodeBalancer struct {
	ID                 int     `json:"id"`
	Label              *string `json:"label"`
	Region             string  `json:"region"`
	Hostname           *string `json:"hostname"`
	IPv4               *string `json:"ipv4"`
	IPv6               *string `json:"ipv6"`
	ClientConnThrottle int     `json:"client_conn_throttle"`

	Tags []string `json:"tags"`
}

func DescribeLoadBalancer(lb *linodego.NodeBalancer) *LinodeNodeBalancer {
	return &LinodeNodeBalancer{
		ID:                 lb.ID,
		Label:              lb.Label,
		Region:             lb.Region,
		Hostname:           lb.Hostname,
		IPv4:               lb.IPv4,
		IPv6:               lb.IPv6,
		ClientConnThrottle: lb.ClientConnThrottle,
		Tags:               lb.Tags,
	}
}
