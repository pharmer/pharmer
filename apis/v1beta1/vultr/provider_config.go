package vultr_config

import (
	"encoding/json"

	. "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	VultrProviderGroupName  = "vultrproviderconfig"
	VultrProviderKind       = "VultrClusterProviderConfig"
	VultrProviderApiVersion = "v1alpha1"
)

type VultrMachineProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Roles  []MachineRole `json:"roles,omitempty"`
	Region string        `json:"region,omitempty"`
	Plan   string        `json:"plan,omitempty"`
	Image  string        `json:"image,omitempty"`
	Pubkey string        `json:"pubkey,omitempty"`
}

type VultrClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func SetVultrClusterProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	conf := &VultrMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VultrProviderGroupName + "/" + VultrProviderApiVersion,
			Kind:       VultrProviderKind,
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

func SetVultrClusterProviderStatus(cluster *clusterapi.Cluster) error {
	conf := &VultrClusterProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: VultrProviderGroupName + "/" + VultrProviderApiVersion,
			Kind:       VultrProviderKind,
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
