package digitalocean

import (
	"github.com/digitalocean/godo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
