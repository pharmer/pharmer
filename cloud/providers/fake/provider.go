package fake

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
)

type ClusterManager struct {
	cf  context.Factory
	cfg *api.PharmerConfig
}

var _ cloud.ClusterProvider = &ClusterManager{}

const (
	UID = "fake"
)

func init() {
	cloud.RegisterCloudProvider(UID, func(ctx context.Context) (cloud.Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.Interface {
	return &ClusterManager{}
}

func (cm *ClusterManager) Clusters() cloud.ClusterProvider {
	return cm
}

func (cm *ClusterManager) Credentials() cloud.CredentialProvider {
	return cm
}

func (p *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	p.runFakeJob("cluster create")
	return nil
}

func (p *ClusterManager) Scale(req *proto.ClusterReconfigureRequest) error {
	p.runFakeJob("cluster scale")
	return nil
}

func (p *ClusterManager) Delete(req *proto.ClusterDeleteRequest) error {
	p.runFakeJob("cluster delete")
	return nil
}

func (p *ClusterManager) SetVersion(req *proto.ClusterReconfigureRequest) error {
	p.runFakeJob("cluster set version")
	return nil
}

func (p *ClusterManager) UploadStartupConfig() error {
	return nil
}

func (p *ClusterManager) GetInstance(md *api.InstanceMetadata) (*api.Instance, error) {
	return &api.Instance{}, nil
}

func (p *ClusterManager) MatchInstance(i *api.Instance, md *api.InstanceMetadata) bool {
	return true
}

func (p *ClusterManager) runFakeJob(requestType string) {
	//c.Logger().Infof("starting %v job", requestType)
	//for i := 1; i <= 10; i++ {
	//	c.Logger().Info(fmt.Sprint("Job completed: ", i*10, "%"))
	//	time.Sleep(time.Second * 3)
	//}
}
