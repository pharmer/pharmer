package fake

import (
	"context"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	core "k8s.io/api/core/v1"
)

type ClusterManager struct {
	cfg *api.PharmerConfig
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

func (cm *ClusterManager) DefaultSpec(in *api.Cluster) (*api.Cluster, error) {
	return in, nil
}

func (cm *ClusterManager) CreateMasterNodeGroup(cluster *api.Cluster) (*api.NodeGroup, error) {
	return nil, nil
}

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	return nil, UnsupportedOperation
}

func (cm *ClusterManager) Check(in *api.Cluster) (string, error) {
	return "", UnsupportedOperation
}

func (cm *ClusterManager) IsValid(cluster *api.Cluster) (bool, error) {
	return false, UnsupportedOperation
}

func (cm *ClusterManager) UploadStartupConfig() error {
	return nil
}

func (cm *ClusterManager) GetInstance(md *api.NodeStatus) (*api.Node, error) {
	return &api.Node{}, nil
}

func (cm *ClusterManager) MatchInstance(i *api.Node, md *api.NodeStatus) bool {
	return true
}

func (cm *ClusterManager) runFakeJob(requestType string) {
	//c.Logger().Infof("starting %v job", requestType)
	//for i := 1; i <= 10; i++ {
	//	c.Logger().Info(fmt.Sprint("Job completed: ", i*10, "%"))
	//	time.Sleep(time.Second * 3)
	//}
}

func (cm *ClusterManager) AssignSSHConfig(cluster *api.Cluster, node *core.Node, cfg *api.SSHConfig) error {
	return UnsupportedOperation
}
