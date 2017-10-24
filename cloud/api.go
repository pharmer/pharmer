package cloud

import (
	"errors"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	core "k8s.io/api/core/v1"
)

var InstanceNotFound = errors.New("Instance not found")
var UnsupportedOperation = errors.New("Unsupported operation")

type Interface interface {
	DefaultSpec(cluster *api.Cluster) (*api.Cluster, error)
	CreateMasterNodeGroup(cluster *api.Cluster) (*api.NodeGroup, error)
	Apply(in *api.Cluster, dryRun bool) ([]api.Action, error)
	IsValid(cluster *api.Cluster) (bool, error)
	Check(cluster *api.Cluster) (string, error)

	// IsValid(cluster *api.Cluster) (bool, error)
	// Delete(req *proto.ClusterDeleteRequest) error
	// SetVersion(req *proto.ClusterReconfigureRequest) error
	// Scale(req *proto.ClusterReconfigureRequest) error
	// GetInstance(md *api.InstanceStatus) (*api.Instance, error)
}

type NodeGroupManager interface {
	Apply(dryRun bool) (acts []api.Action, err error)
	AddNodes(count int64) error
	DeleteNodes(nodes []core.Node) error
}

type InstanceManager interface {
	CreateInstance(name, token string, ng *api.NodeGroup) (*api.SimpleNode, error)
	DeleteInstanceByProviderID(providerID string) error
}

type UpgradeManager interface {
	GetAvailableUpgrades() ([]api.Upgrade, error)
	PrintAvailableUpgrades([]api.Upgrade)
	Apply(dryRun bool) ([]api.Action, error)
	MasterUpgrade() error
	NodeGroupUpgrade(ng *api.NodeGroup) error
}

type SSHExecutor interface {
	ExecuteSSHCommand(command string, instance *core.Node) (string, error)
}
