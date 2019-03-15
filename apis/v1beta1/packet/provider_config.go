package packet_config

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	PacketProviderGroupName  = "Packetproviderconfig"
	PacketProviderKind       = "PacketClusterProviderConfig"
	PacketProviderApiVersion = "v1alpha1"
)

type PacketMachineProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Plan         string `json:"region,omitempty"`
	SpotInstance string `json:"type,omitempty"`
}

type PacketClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func SetPacketClusterProviderConfig(cluster *clusterapi.Cluster) error {
	conf := &PacketMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: PacketProviderGroupName + "/" + PacketProviderApiVersion,
			Kind:       PacketProviderKind,
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

func SetPacketClusterProviderStatus(cluster *clusterapi.Cluster) error {
	conf := &PacketClusterProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: PacketProviderGroupName + "/" + PacketProviderApiVersion,
			Kind:       PacketProviderKind,
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
