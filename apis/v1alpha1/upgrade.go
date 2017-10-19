package v1alpha1

import (
	"github.com/appscode/mergo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//https://github.com/kubernetes/kubernetes/blob/aa1dc9db3532dfbf09e45c8e3786a648cd217417/cmd/kubeadm/app/phases/upgrade/compute.go#L28
type Upgrade struct {
	metav1.TypeMeta `json:",inline,omitempty,omitempty"`

	Description string
	Before      ClusterState
	After       ClusterState
}

// ClusterState describes the state of certain versions for a cluster
type ClusterState struct {
	// KubeVersion describes the version of the Kubernetes API Server, Controller Manager, Scheduler and Proxy.
	KubeVersion string
	// DNSVersion describes the version of the kube-dns images used and manifest version
	DNSVersion string
	// MasterKubeadmVersion describes the version of the kubeadm CLI
	KubeadmVersion string
	// KubeletVersions is a map with a version number linked to the amount of kubelets running that version in the cluster
	KubeletVersions map[string]uint16
}

var _ runtime.Object = &Action{}

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

func (u *Upgrade) DeepCopyObject() runtime.Object {
	if u == nil {
		return u
	}
	out := new(Upgrade)
	mergo.MergeWithOverwrite(out, u)
	return out
}
