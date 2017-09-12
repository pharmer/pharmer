package cloud

import (
	"errors"

	"github.com/appscode/pharmer/api"
)

var InstanceNotFound = errors.New("Instance not found")
var UnsupportedOperation = errors.New("Unsupported operation")

type Interface interface {
	DefaultSpec(cluster *api.Cluster) (*api.Cluster, error)
	CreateMasterNodeGroup(cluster *api.Cluster) (*api.NodeGroup, error)
	Apply(cluster *api.Cluster, dryRun bool) error
	IsValid(cluster *api.Cluster) (bool, error)

	// IsValid(cluster *api.Cluster) (bool, error)
	// Delete(req *proto.ClusterDeleteRequest) error
	// SetVersion(req *proto.ClusterReconfigureRequest) error
	// Scale(req *proto.ClusterReconfigureRequest) error
	// GetInstance(md *api.InstanceStatus) (*api.Instance, error)
}
