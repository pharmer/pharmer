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
	api "pharmer.dev/pharmer/apis/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
)

const (
	GCEProviderGroupName   = "gceproviderconfig"
	GCEClusterProviderKind = "GCEClusterProviderSpec"
	GCEMachineProviderKind = "GCEMachineProviderSpec"
	GCEProviderAPIVersion  = "v1alpha1"
)

// GCEMachineProviderSpec is the Schema for the gcemachineproviderconfigs API
// +k8s:openapi-gen=true
type GCEMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Roles             []api.MachineRole `json:"roles,omitempty"`

	Zone        string `json:"zone"`
	MachineType string `json:"machineType"`

	// The name of the OS to be installed on the machine.
	OS    string `json:"os,omitempty"`
	Disks []Disk `json:"disks,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GCEClusterProviderSpec is the Schema for the gceclusterproviderconfigs API
// +k8s:openapi-gen=true
type GCEClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Project string `json:"project"`

	// CAKeyPair is the key pair for CA certs.
	CAKeyPair KeyPair `json:"caKeyPair,omitempty"`

	// EtcdCAKeyPair is the key pair for etcd.
	EtcdCAKeyPair KeyPair `json:"etcdCAKeyPair,omitempty"`

	// FrontProxyCAKeyPair is the key pair for the front proxy.
	FrontProxyCAKeyPair KeyPair `json:"frontProxyCAKeyPair,omitempty"`

	// SAKeyPair is the service account key pair.
	SAKeyPair KeyPair `json:"saKeyPair,omitempty"`

	// AdminKubeconfig generated using the certificates part of the spec
	// do not move to status, since it uses on disk ca certs, which causes issues during regeneration
	AdminKubeconfig string `json:"adminKubeconfig,omitempty"`

	// DiscoveryHashes generated using the certificates part of the spec, used by master and nodes bootstrapping
	// this never changes until ca is rotated
	// do not move to status, since it uses on disk ca certs, which causes issues during regeneration
	DiscoveryHashes []string `json:"discoveryHashes,omitempty"`

	// ClusterConfiguration holds the cluster-wide information used during a
	// kubeadm init call.
	ClusterConfiguration kubeadmv1beta1.ClusterConfiguration `json:"clusterConfiguration,omitempty"`
}

// KeyPair is how operators can supply custom keypairs for kubeadm to use.
type KeyPair struct {
	// base64 encoded cert and key
	Cert []byte `json:"cert"`
	Key  []byte `json:"key"`
}

type GCEClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type Disk struct {
	InitializeParams DiskInitializeParams `json:"initializeParams"`
}

type DiskInitializeParams struct {
	DiskSizeGb int64  `json:"diskSizeGb"`
	DiskType   string `json:"diskType"`
}
