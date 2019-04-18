package linode_config

import (
	"encoding/json"

	"github.com/linode/linodego"
	. "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	LinodeProviderGroupName  = "linodeproviderconfig"
	LinodeProviderKind       = "LinodeClusterProviderConfig"
	LinodeProviderApiVersion = "v1alpha1"
)

type LinodeMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Roles  []MachineRole `json:"roles,omitempty"`
	Region string        `json:"region,omitempty"`
	Type   string        `json:"type,omitempty"`
	Image  string        `json:"image,omitempty"`
	Pubkey string        `json:"pubkey,omitempty"`
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

//func (c *Cluster) SetLinodeProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
func SetLinodeClusterProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	conf := &LinodeMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: LinodeProviderGroupName + "/" + LinodeProviderApiVersion,
			Kind:       LinodeProviderKind,
		},
	}
	bytes, err := json.Marshal(conf)
	if err != nil {
		return err

	}
	cluster.Spec.ProviderSpec = clusterapi.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: bytes,
		},
	}
	return nil
}

func SetLinodeClusterProviderStatus(cluster *clusterapi.Cluster) error {
	conf := &LinodeClusterProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: LinodeProviderGroupName + "/" + LinodeProviderApiVersion,
			Kind:       LinodeProviderKind,
		},
	}
	bytes, err := json.Marshal(conf)
	if err != nil {
		return err

	}
	cluster.Status.ProviderStatus = &runtime.RawExtension{
		Raw: bytes,
	}
	return nil
}
