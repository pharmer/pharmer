package cloud

import (
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
)

var (
	ErrNotFound       = errors.New("node not found")
	ErrNotImplemented = errors.New("not implemented")
	ErrNoMasterNG     = errors.New("cluster has no master NodeGroup")
)

type Interface interface {
	SSHGetter
	ProviderKubeConfig
	GetDefaultNodeSpec(cluster *api.Cluster, sku string) (api.NodeSpec, error)
	SetDefaults(in *api.Cluster) error
	Apply(in *api.Cluster, dryRun bool) ([]api.Action, error)
	IsValid(cluster *api.Cluster) (bool, error)
	SetOwner(owner string)
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
	CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error)
	DeleteInstanceByProviderID(providerID string) error
}

type UpgradeManager interface {
	GetAvailableUpgrades() ([]*api.Upgrade, error)
	PrintAvailableUpgrades([]*api.Upgrade)
	Apply(dryRun bool) ([]api.Action, error)
	MasterUpgrade() error
	NodeGroupUpgrade(ng *api.NodeGroup) error
}

type ProviderKubeConfig interface {
	GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error)
}

type HookFunc func() error
