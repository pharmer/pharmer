package packet

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// ClusterConfigFromProviderSpec unmarshals a provider config into an Linode Cluster type
func ClusterConfigFromProviderSpec(providerConfig clusterv1.ProviderSpec) (*PacketClusterProviderSpec, error) {
	var config PacketClusterProviderSpec
	if providerConfig.Value == nil {
		return &config, nil
	}

	if err := json.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ClusterStatusFromProviderStatus unmarshals a raw extension into an Linode Cluster type
func ClusterStatusFromProviderStatus(extension *runtime.RawExtension) (*PacketClusterProviderStatus, error) {
	if extension == nil {
		return &PacketClusterProviderStatus{}, nil
	}

	status := new(PacketClusterProviderStatus)
	if err := json.Unmarshal(extension.Raw, status); err != nil {
		return nil, err
	}

	return status, nil
}

// MachineSpecFromProviderSpec unmarshals a raw extension into an Linode machine type
func MachineConfigFromProviderSpec(providerConfig clusterv1.ProviderSpec) (*PacketMachineProviderSpec, error) {
	var config PacketMachineProviderSpec
	if providerConfig.Value == nil {
		return &config, nil
	}

	if err := json.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// MachineStatusFromProviderStatus unmarshals a raw extension into an Linode machine type
func MachineStatusFromProviderStatus(extension *runtime.RawExtension) (*PacketMachineProviderStatus, error) {
	if extension == nil {
		return &PacketMachineProviderStatus{}, nil
	}

	status := new(PacketMachineProviderStatus)
	if err := json.Unmarshal(extension.Raw, status); err != nil {
		return nil, err
	}

	return status, nil
}

// EncodeMachineStatus marshals the machine status
func EncodeMachineStatus(status *PacketMachineProviderStatus) (*runtime.RawExtension, error) {
	if status == nil {
		return &runtime.RawExtension{}, nil
	}

	var rawBytes []byte
	var err error

	if rawBytes, err = json.Marshal(status); err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}

// EncodeMachineSpec marshals the machine provider spec.
func EncodeMachineSpec(spec *PacketMachineProviderSpec) (*runtime.RawExtension, error) {
	if spec == nil {
		return &runtime.RawExtension{}, nil
	}

	var rawBytes []byte
	var err error

	if rawBytes, err = json.Marshal(spec); err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}

// EncodeClusterStatus marshals the cluster status.
func EncodeClusterStatus(status *PacketClusterProviderStatus) (*runtime.RawExtension, error) {
	if status == nil {
		return &runtime.RawExtension{}, nil
	}

	var rawBytes []byte
	var err error

	if rawBytes, err = json.Marshal(status); err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}

// EncodeClusterSpec marshals the cluster provider spec.
func EncodeClusterSpec(spec *PacketClusterProviderSpec) (*runtime.RawExtension, error) {
	if spec == nil {
		return &runtime.RawExtension{}, nil
	}

	var rawBytes []byte
	var err error

	if rawBytes, err = json.Marshal(spec); err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}

func SetPacketClusterProviderConfig(cluster *clusterapi.Cluster) error {
	conf := &PacketMachineProviderSpec{
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
