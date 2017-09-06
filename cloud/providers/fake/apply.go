package fake

import "github.com/appscode/pharmer/cloud"

func (cm *ClusterManager) Apply(cluster string, dryRun bool) error {
	return cloud.UnsupportedOperation
}

func (cm *ClusterManager) IsValid(cluster string) (bool, error) {
	return false, cloud.UnsupportedOperation
}
