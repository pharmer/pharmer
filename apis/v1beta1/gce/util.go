package gce

import (
	"encoding/json"

	. "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// ClusterConfigFromProviderSpec unmarshals a provider config into an AWS Cluster type
func ClusterConfigFromProviderSpec(providerConfig clusterv1.ProviderSpec) (*GCEClusterProviderSpec, error) {
	var config GCEClusterProviderSpec
	if providerConfig.Value == nil {
		return &config, nil
	}

	if err := json.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ClusterStatusFromProviderStatus unmarshals a raw extension into an AWS Cluster type
func ClusterStatusFromProviderStatus(extension *runtime.RawExtension) (*GCEClusterProviderStatus, error) {
	if extension == nil {
		return &GCEClusterProviderStatus{}, nil
	}

	status := new(GCEClusterProviderStatus)
	if err := json.Unmarshal(extension.Raw, status); err != nil {
		return nil, err
	}

	return status, nil
}

// MachineSpecFromProviderSpec unmarshals a raw extension into an AWS machine type
func MachineConfigFromProviderSpec(providerConfig clusterv1.ProviderSpec) (*GCEMachineProviderSpec, error) {
	var config GCEMachineProviderSpec
	if providerConfig.Value == nil {
		return &config, nil
	}

	if err := json.Unmarshal(providerConfig.Value.Raw, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// EncodeMachineSpec marshals the machine provider spec.
func EncodeMachineSpec(spec *GCEMachineProviderSpec) (*runtime.RawExtension, error) {
	if spec == nil {
		return &runtime.RawExtension{}, nil
	}

	var rawBytes []byte
	var err error

	//  TODO: use apimachinery conversion https://godoc.org/k8s.io/apimachinery/pkg/runtime#Convert_runtime_Object_To_runtime_RawExtension
	if rawBytes, err = json.Marshal(spec); err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}

// EncodeClusterStatus marshals the cluster status.
func EncodeClusterStatus(status *GCEClusterProviderStatus) (*runtime.RawExtension, error) {
	if status == nil {
		return &runtime.RawExtension{}, nil
	}

	var rawBytes []byte
	var err error

	//  TODO: use apimachinery conversion https://godoc.org/k8s.io/apimachinery/pkg/runtime#Convert_runtime_Object_To_runtime_RawExtension
	if rawBytes, err = json.Marshal(status); err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}

// EncodeClusterSpec marshals the cluster provider spec.
func EncodeClusterSpec(spec *GCEClusterProviderSpec) (*runtime.RawExtension, error) {
	if spec == nil {
		return &runtime.RawExtension{}, nil
	}

	var rawBytes []byte
	var err error

	//  TODO: use apimachinery conversion https://godoc.org/k8s.io/apimachinery/pkg/runtime#Convert_runtime_Object_To_runtime_RawExtension
	if rawBytes, err = json.Marshal(spec); err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: rawBytes,
	}, nil
}

// SetGCEClusterProvidreConfig sets default gce cluster providerSpec
func SetGCEclusterProviderConfig(cluster *clusterv1.Cluster, config *ClusterConfig) error {
	conf := &GCEClusterProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GCEProviderGroupName + "/" + GCEProviderApiVersion,
			Kind:       GCEClusterProviderKind,
		},
		Project: config.Cloud.Project,
	}

	rawConf, err := EncodeClusterSpec(conf)
	if err != nil {
		return errors.Wrap(err, "failed to encode cluster provider spec")
	}

	cluster.Spec.ProviderSpec.Value = rawConf

	return nil
}
