package cloud

import (
	"errors"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	core "k8s.io/api/core/v1"
)

var ErrNotFound = errors.New("node not found")
var ErrNotImplemented = errors.New("not implemented")

type Interface interface {
	SSHGetter
	DefaultSpec(cluster *api.Cluster) (*api.Cluster, error)
	CreateMasterNodeGroup(cluster *api.Cluster) (*api.NodeGroup, error)
	Apply(in *api.Cluster, dryRun bool) ([]api.Action, error)
	IsValid(cluster *api.Cluster) (bool, error)
	// GetAdminClient() (kubernetes.Interface, error)

	// IsValid(cluster *api.Cluster) (bool, error)
	// Delete(req *proto.ClusterDeleteRequest) error
	// SetVersion(req *proto.ClusterReconfigureRequest) error
	// Scale(req *proto.ClusterReconfigureRequest) error
	// GetInstance(md *api.InstanceStatus) (*api.Instance, error)
}

type SSHGetter interface {
	GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error)
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

type HookFunc func() error
