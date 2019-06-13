package gce

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManager adds all Controllers to the Manager
func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	return nil
}
