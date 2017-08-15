package api

import (
	"sync"

	"github.com/appscode/pharmer/storage"
)

type InstanceMetadata struct {
	ExternalID string
	Name       string
	ExternalIP string
	InternalIP string
}

type KubernetesInstance struct {
	// TODO(tamal): May be embed InstanceMetadata
	ExternalID string
	Name       string
	ExternalIP string
	InternalIP string

	PHID           string
	ExternalStatus string
	SKU            string
	Role           string
	Status         string
}

// Embed this context in actual providers.
type ClusterInstances struct {
	m sync.Mutex

	KubernetesPHID string
	Instances      []*KubernetesInstance

	matches func(i *KubernetesInstance, md *InstanceMetadata) bool
}

func (ins *ClusterInstances) Save() error {
	return nil
}

// Set ctx.Name (required)
// Set ctx.ContextVersion (optional) to load specific version
func (ins *ClusterInstances) Load() error {

	return nil
}

// Does not modify ctx.NumNodes; Reduce ctx.NumNodes separately
func (ins *ClusterInstances) FindInstance(md *InstanceMetadata) (*KubernetesInstance, bool) {
	for _, i := range ins.Instances {
		if ins.matches(i, md) {
			return i, true
		}
	}
	return nil, false
}

// Does not modify ctx.NumNodes; Reduce ctx.NumNodes separately
func (ins *ClusterInstances) DeleteInstance(instance *KubernetesInstance) (*KubernetesInstance, error) {
	// TODO(tamal): FixIt!
	//updates := &storage.KubernetesInstance{Status: storage.KubernetesInstanceStatus_Deleted}
	//cond := &storage.KubernetesInstance{PHID: instance.PHID}
	//if _, err := ins.Store().Engine.Update(updates, cond); err != nil {
	//	return nil, errors.FromErr(err).WithContext(ins).Err()
	//} else {
	instance.Status = storage.KubernetesInstanceStatus_Deleted
	return instance, nil
	//}
}
