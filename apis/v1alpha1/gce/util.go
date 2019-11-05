/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package gce

import (
	"encoding/json"

	"pharmer.dev/pharmer/cloud/utils/certificates"

	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// ClusterConfigFromProviderSpec unmarshals a provider config into an AWS Cluster type
func ClusterConfigFromProviderSpec(providerConfig clusterapi.ProviderSpec) (*GCEClusterProviderSpec, error) {
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
func MachineConfigFromProviderSpec(providerConfig clusterapi.ProviderSpec) (*GCEMachineProviderSpec, error) {
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
func SetGCEclusterProviderConfig(cluster *clusterapi.Cluster, project string, certs *certificates.Certificates) error {
	conf := &GCEClusterProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GCEProviderGroupName + "/" + GCEProviderAPIVersion,
			Kind:       GCEClusterProviderKind,
		},
		Project: project,
		CAKeyPair: KeyPair{
			Cert: cert.EncodeCertPEM(certs.CACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(certs.CACert.Key),
		},
		EtcdCAKeyPair: KeyPair{
			Cert: cert.EncodeCertPEM(certs.EtcdCACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(certs.EtcdCACert.Key),
		},
		FrontProxyCAKeyPair: KeyPair{
			Cert: cert.EncodeCertPEM(certs.FrontProxyCACert.Cert),
			Key:  cert.EncodePrivateKeyPEM(certs.FrontProxyCACert.Key),
		},
		SAKeyPair: KeyPair{
			Cert: cert.EncodeCertPEM(certs.ServiceAccountCert.Cert),
			Key:  cert.EncodePrivateKeyPEM(certs.ServiceAccountCert.Key),
		},
	}

	rawConf, err := EncodeClusterSpec(conf)
	if err != nil {
		return errors.Wrap(err, "failed to encode cluster provider spec")
	}

	cluster.Spec.ProviderSpec.Value = rawConf

	return nil
}
