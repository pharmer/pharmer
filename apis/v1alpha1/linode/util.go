package linode

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// ClusterConfigFromProviderSpec unmarshals a provider config into an Linode Cluster type
func ClusterConfigFromProviderSpec(providerConfig clusterapi.ProviderSpec) (*LinodeClusterProviderSpec, error) {
	var config LinodeClusterProviderSpec
	if providerConfig.Value == nil {
		return &config, nil
	}

	if err := json.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ClusterStatusFromProviderStatus unmarshals a raw extension into an Linode Cluster type
func ClusterStatusFromProviderStatus(extension *runtime.RawExtension) (*LinodeClusterProviderStatus, error) {
	if extension == nil {
		return &LinodeClusterProviderStatus{}, nil
	}

	status := new(LinodeClusterProviderStatus)
	if err := json.Unmarshal(extension.Raw, status); err != nil {
		return nil, err
	}

	return status, nil
}

// MachineSpecFromProviderSpec unmarshals a raw extension into an Linode machine type
func MachineConfigFromProviderSpec(providerConfig clusterapi.ProviderSpec) (*LinodeMachineProviderSpec, error) {
	var config LinodeMachineProviderSpec
	if providerConfig.Value == nil {
		return &config, nil
	}

	if err := json.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// MachineStatusFromProviderStatus unmarshals a raw extension into an Linode machine type
func MachineStatusFromProviderStatus(extension *runtime.RawExtension) (*LinodeMachineProviderStatus, error) {
	if extension == nil {
		return &LinodeMachineProviderStatus{}, nil
	}

	status := new(LinodeMachineProviderStatus)
	if err := json.Unmarshal(extension.Raw, status); err != nil {
		return nil, err
	}

	return status, nil
}

// EncodeMachineStatus marshals the machine status
func EncodeMachineStatus(status *LinodeMachineProviderStatus) (*runtime.RawExtension, error) {
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
func EncodeMachineSpec(spec *LinodeMachineProviderSpec) (*runtime.RawExtension, error) {
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
func EncodeClusterStatus(status *LinodeClusterProviderStatus) (*runtime.RawExtension, error) {
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
func EncodeClusterSpec(spec *LinodeClusterProviderSpec) (*runtime.RawExtension, error) {
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

//func (c *Cluster) SetLinodeProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
func SetLinodeClusterProviderConfig(cluster *clusterapi.Cluster) error {
	conf := &LinodeMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: LinodeProviderGroupName + "/" + LinodeProviderAPIVersion,
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
