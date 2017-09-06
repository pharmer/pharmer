package fake

import (
	"context"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

type ClusterManager struct {
	cfg *api.PharmerConfig
}

var _ cloud.ClusterManager = &ClusterManager{}

const (
	UID = "fake"
)

func init() {
	cloud.RegisterCloudManager(UID, func(ctx context.Context) (cloud.ClusterManager, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.ClusterManager {
	return &ClusterManager{}
}

func (cm *ClusterManager) DefaultSpec(in *api.Cluster) (*api.Cluster, error) {
	return in, nil
}

func (cm *ClusterManager) CreateMasterInstanceGroup(cluster *api.Cluster) (*api.InstanceGroup, error) {
	return nil, nil
}

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) error {
	return cloud.UnsupportedOperation
}

func (cm *ClusterManager) IsValid(cluster string) (bool, error) {
	return false, cloud.UnsupportedOperation
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

func (cm *ClusterManager) GetInstance(md *api.InstanceStatus) (*api.Instance, error) {
	return &api.Instance{}, nil
}

func (cm *ClusterManager) MatchInstance(i *api.Instance, md *api.InstanceStatus) bool {
	return true
}

func (cm *ClusterManager) runFakeJob(requestType string) {
	//c.Logger().Infof("starting %v job", requestType)
	//for i := 1; i <= 10; i++ {
	//	c.Logger().Info(fmt.Sprint("Job completed: ", i*10, "%"))
	//	time.Sleep(time.Second * 3)
	//}
}
