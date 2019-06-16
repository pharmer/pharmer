package cloud

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	ErrNotFound       = errors.New("node not found")
	ErrNotImplemented = errors.New("not implemented")
	ErrNoMasterNG     = errors.New("Cluster has no master NodeGroup")
)

type Interface interface {
	CloudManagerInterface

	CreateCredentials(kc kubernetes.Interface) error

	//GetConnector() ClusterApiProviderComponent

	InitializeMachineActuator(mgr manager.Manager) error

	AddToManager(m manager.Manager) error

	//SSHGetter
	ProviderKubeConfig

	GetCloudConnector() error

	ApplyDelete() error
	// only managed providers
	ApplyScale() error
	SetDefaultCluster() error
	GetDefaultMachineProviderSpec(sku string, role api.MachineRole) (clusterapi.ProviderSpec, error)

	NewMasterTemplateData(machine *clusterapi.Machine, token string, td TemplateData) TemplateData
	NewNodeTemplateData(machine *clusterapi.Machine, token string, td TemplateData) TemplateData

	PrepareCloud() error
	EnsureMaster() error

	GetMasterSKU(totalNodes int32) string

	GetClusterAPIComponents() (string, error)
}

type SSHGetter interface {
	GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error)
}

type NodeGroupManager interface {
	//	Apply(dryRun bool) (acts []api.Action, err error)
	//	AddNodes(count int64) error
	//	DeleteNodes(nodes []core.Node) error
}

// TODO: change name
type ClusterApiProviderComponent interface {
	CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error
	GetControllerManager() (string, error)
}

type UpgradeManager interface {
	GetAvailableUpgrades() ([]*api.Upgrade, error)
	PrintAvailableUpgrades([]*api.Upgrade)
	Apply(dryRun bool) ([]api.Action, error)
	MasterUpgrade(oldMachine *clusterapi.Machine, newMachine *clusterapi.Machine) error
	NodeUpgrade(oldMachine *clusterapi.Machine, newMachine *clusterapi.Machine) error
}

type ProviderKubeConfig interface {
	GetKubeConfig() (*api.KubeConfig, error)
}

type HookFunc func() error
