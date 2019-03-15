package fake

import (
	"context"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	core "k8s.io/api/core/v1"
)

type ClusterManager struct {
	cfg   *api.PharmerConfig
	owner string
}

var _ Interface = &ClusterManager{}

const (
	UID = "fake"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
	return &ClusterManager{}
}

func (cm *ClusterManager) SetDefaults(in *api.Cluster) error {
	return nil
}

func (cm *ClusterManager) SetOwner(owner string) {
	cm.owner = owner
}

func (cm *ClusterManager) GetDefaultNodeSpec(cluster *api.Cluster, sku string) (api.NodeSpec, error) {
	return api.NodeSpec{}, nil
}

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	return nil, ErrNotImplemented
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, ErrNotImplemented
}

func (cm *ClusterManager) UploadStartupConfig() error {
	return nil
}

func (cm *ClusterManager) runFakeJob(requestType string) {
	//c.Logger().Infof("starting %v job", requestType)
	//for i := 1; i <= 10; i++ {
	//	c.Logger().Info(fmt.Sprint("Job completed: ", i*10, "%"))
	//	time.Sleep(time.Second * 3)
	//}
}

func (cm *ClusterManager) GetSSHConfig(cluster *api.Cluster, node *core.Node) (*api.SSHConfig, error) {
	return nil, ErrNotImplemented
}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	return nil, ErrNotImplemented
}
