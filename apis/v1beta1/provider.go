package v1beta1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	DigitalOceanProviderGroupName="digitaloceanproviderconfig"
	DigitalOceanProviderKind="digitaloceanproviderconfig"
)
// DigitalOceanMachineProviderConfig contains Config for DigitalOcean machines.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DigitalOceanMachineProviderConfig struct {
	metav1.TypeMeta `json:",inline"`

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


func (c *Kube) DigitalOceanProviderConfig() *ClusterProviderConfig {
	//providerConfig providerConfig
	raw := c.Spec.ClusterAPI.Spec.ProviderConfig.Value.Raw
	providerConfig := &ClusterProviderConfig{}
	err := json.Unmarshal(raw, providerConfig)
	if err != nil {
		fmt.Println("Unable to unmarshal provider config: %v", err)
	}
	return providerConfig
}

func (c *Kube) SetDigitalOceanProviderConfig(cluster *clusterapi.Cluster, config *ClusterProviderConfig) error {
	bytes, err := json.Marshal(config)
	if err != nil {
		fmt.Println("Unable to marshal provider config: %v", err)
		return err
	}
	cluster.Spec.ProviderConfig = clusterapi.ProviderConfig{
		Value: &runtime.RawExtension{
			Raw: bytes,
		},
	}
	return nil
}
