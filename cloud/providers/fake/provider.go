package fake

import (
	"context"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
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

func (cm *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	cm.runFakeJob("cluster create")
	return nil
}

func (cm *ClusterManager) Scale(req *proto.ClusterReconfigureRequest) error {
	cm.runFakeJob("cluster scale")
	return nil
}

func (cm *ClusterManager) Delete(req *proto.ClusterDeleteRequest) error {
	cm.runFakeJob("cluster delete")
	return nil
}

func (cm *ClusterManager) SetVersion(req *proto.ClusterReconfigureRequest) error {
	cm.runFakeJob("cluster set version")
	return nil
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
