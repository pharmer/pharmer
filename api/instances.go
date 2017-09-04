package api

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstanceGroup struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	ObjectMeta      `json:"metadata,omitempty"`
	Spec            InstanceGroupSpec   `json:"spec,omitempty"`
	Status          InstanceGroupStatus `json:"status,omitempty"`
}

type InstanceGroupSpec struct {
	SKU              string `json:"sku,omitempty"`
	Count            int64  `json:"count,omitempty"`
	UseSpotInstances bool   `json:"useSpotInstances,omitempty"`
}

type InstanceGroupStatus struct {
}

type InstanceMetadata struct {
	ExternalID string
	Name       string
	ExternalIP string
	InternalIP string
}

type Instance struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	ObjectMeta      `json:"metadata,omitempty"`
	Spec            InstanceSpec   `json:"spec,omitempty"`
	Status          InstanceStatus `json:"status,omitempty"`
}

type InstanceSpec struct {
	SKU  string
	Role string
}

type InstanceStatus struct {
	ExternalID    string
	ExternalIP    string
	InternalIP    string
	ExternalPhase string
	Phase         string
}

// Embed this context in actual providers.
type ClusterInstances struct {
	m sync.Mutex

	KubernetesPHID string
	Instances      []*Instance

	matches func(i *Instance, md *InstanceMetadata) bool
}

// Does not modify ctx.NumNodes; Reduce ctx.NumNodes separately
func (ins *ClusterInstances) FindInstance(md *InstanceMetadata) (*Instance, bool) {
	for _, i := range ins.Instances {
		if ins.matches(i, md) {
			return i, true
		}
	}
	return nil, false
}

// Does not modify ctx.NumNodes; Reduce ctx.NumNodes separately
func (ins *ClusterInstances) DeleteInstance(instance *Instance) (*Instance, error) {
	// TODO(tamal): FixIt!
	//updates := &KubernetesInstance{Status: InstancePhaseDeleted}
	//cond := &KubernetesInstance{PHID: instance.PHID}
	//if _, err := ins.Store().Engine.Update(updates, cond); err != nil {
	//	return nil, errors.FromErr(err).WithContext(ins).Err()
	//} else {
	instance.Status.Phase = InstancePhaseDeleted
	return instance, nil
	//}
}
