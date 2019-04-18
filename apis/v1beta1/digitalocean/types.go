package digitalocean

import (
	"encoding/json"

	"github.com/digitalocean/godo"
	. "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	DigitalOceanProviderGroupName  = "digitaloceanproviderconfig"
	DigitalOceanProviderKind       = "DigitalOceanProviderConfig"
	DigitalOceanProviderApiVersion = "v1alpha1"
)

type DigitalOceanMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:",inline"`

	Region        string   `json:"region,omitempty"`
	Size          string   `json:"size,omitempty"`
	Image         string   `json:"image,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	SSHPublicKeys []string `json:"sshPublicKeys,omitempty"`

	PrivateNetworking bool `json:"private_networking,omitempty"`
	Backups           bool `json:"backups,omitempty"`
	IPv6              bool `json:"ipv6,omitempty"`
	Monitoring        bool `json:"monitoring,omitempty"`
}

type DigitalOceanMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	InstanceID     int    `json:"instanceID"`
	InstanceStatus string `json:"instanceStatus"`
}

type DigitalOceanClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type DigitalOceanClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	APIServerLB *DigitalOceanLoadBalancer `json:"apiServerLb,omitempty"`
}

type DigitalOceanLoadBalancer struct {
	ID                  string                `json:"id,omitempty"`
	Name                string                `json:"name,omitempty"`
	IP                  string                `json:"ip,omitempty"`
	Algorithm           string                `json:"algorithm,omitempty"`
	Status              string                `json:"status,omitempty"`
	Created             string                `json:"created_at,omitempty"`
	ForwardingRules     []godo.ForwardingRule `json:"forwarding_rules,omitempty"`
	HealthCheck         *godo.HealthCheck     `json:"health_check,omitempty"`
	StickySessions      *godo.StickySessions  `json:"sticky_sessions,omitempty"`
	Region              string                `json:"region,omitempty"`
	RedirectHttpToHttps bool                  `json:"redirect_http_to_https,omitempty"`
}

//func (c *Cluster) SetLinodeProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
func SetDigitalOceanClusterProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	conf := &DigitalOceanClusterProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: DigitalOceanProviderGroupName + "/" + DigitalOceanProviderApiVersion,
			Kind:       DigitalOceanProviderKind,
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

func SetDigitalOceanClusterProviderStatus(cluster *clusterapi.Cluster) error {
	conf := &DigitalOceanClusterProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: DigitalOceanProviderGroupName + "/" + DigitalOceanProviderApiVersion,
			Kind:       DigitalOceanProviderKind,
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

func DescribeLoadBalancer(lb *godo.LoadBalancer) *DigitalOceanLoadBalancer {
	return &DigitalOceanLoadBalancer{
		ID:                  lb.ID,
		Name:                lb.Name,
		IP:                  lb.IP,
		Algorithm:           lb.Algorithm,
		Status:              lb.Status,
		Created:             lb.Created,
		ForwardingRules:     lb.ForwardingRules,
		HealthCheck:         lb.HealthCheck,
		StickySessions:      lb.StickySessions,
		Region:              lb.Region.Slug,
		RedirectHttpToHttps: lb.RedirectHttpToHttps,
	}
}
