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
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//https://github.com/kubernetes/kubernetes/blob/aa1dc9db3532dfbf09e45c8e3786a648cd217417/cmd/kubeadm/app/phases/upgrade/compute.go#L28
type Upgrade struct {
	metav1.TypeMeta `json:",inline"`

	Description string       `json:"description"`
	Before      ClusterState `json:"before"`
	After       ClusterState `json:"after"`
}

// ClusterState describes the state of certain versions for a cluster
type ClusterState struct {
	// KubeVersion describes the version of the Kubernetes API Server, Controller Manager, Scheduler and Proxy.
	KubeVersion string `json:"kubeVersion"`
	// DNSVersion describes the version of the kube-dns images used and manifest version
	DNSVersion string `json:"dnsVersion"`
	// MasterKubeadmVersion describes the version of the kubeadm CLI
	KubeadmVersion string `json:"kubeadmVersion"`
	// KubeletVersions is a map with a version number linked to the amount of kubelets running that version in the cluster
	KubeletVersions map[string]uint32 `json:"kubeletVersions"`
}

// CanUpgradeKubelets returns whether an upgrade of any kubelet in the cluster is possible
func (u *Upgrade) CanUpgradeKubelets() bool {
	// If there are multiple different versions now, an upgrade is possible (even if only for a subset of the nodes)
	if len(u.Before.KubeletVersions) > 1 {
		return true
	}
	// Don't report something available for upgrade if we don't know the current state
	if len(u.Before.KubeletVersions) == 0 {
		return false
	}

	// if the same version number existed both before and after, we don't have to upgrade it
	_, sameVersionFound := u.Before.KubeletVersions[u.After.KubeVersion]
	return !sameVersionFound
}
