package cloud

import (
	apiv1 "github.com/pharmer/pharmer/apis/v1"
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
	GetDefaultNodeSpec(cluster *apiv1.Cluster, sku string) (apiv1.NodeSpec, error)
	//GetDefaultMachineSpec(cluster *api.Cluster, sku string) ()
	SetDefaults(in *api.Cluster) error
	SetDefaultCluster(in *apiv1.Cluster, conf *apiv1.ClusterProviderConfig) error
	Apply(in *apiv1.Cluster, dryRun bool) ([]apiv1.Action, error)
	IsValid(cluster *api.Cluster) (bool, error)
	// GetAdminClient() (kubernetes.Interface, error)

	// IsValid(cluster *api.Cluster) (bool, error)
	// Delete(req *proto.ClusterDeleteRequest) error
	// SetVersion(req *proto.ClusterReconfigureRequest) error
	// Scale(req *proto.ClusterReconfigureRequest) error
	// GetInstance(md *api.InstanceStatus) (*api.Instance, error)
}

type SSHGetter interface {
	GetSSHConfig(cluster *apiv1.Cluster, node *core.Node) (*apiv1.SSHConfig, error)
}

type NodeGroupManager interface {
	Apply(dryRun bool) (acts []api.Action, err error)
	AddNodes(count int64) error
	DeleteNodes(nodes []core.Node) error
}

type InstanceManager interface {
	//CreateInstance(name, token string, ng *api.NodeGroup) (*api.NodeInfo, error)
	DeleteInstanceByProviderID(providerID string) error
}

type UpgradeManager interface {
	GetAvailableUpgrades() ([]*api.Upgrade, error)
	PrintAvailableUpgrades([]*api.Upgrade)
	Apply(dryRun bool) ([]api.Action, error)
	MasterUpgrade() error
	NodeGroupUpgrade(ng *api.NodeGroup) error
}

type HookFunc func() error
