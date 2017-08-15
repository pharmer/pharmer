package fake

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func init() {
	cloud.RegisterCloudProvider("fake", new(kubeProvider))
}

type kubeProvider struct {
}

var _ cloud.Provider = &kubeProvider{}

func (cluster *kubeProvider) Create(ctx *api.Cluster, req *proto.ClusterCreateRequest) error {
	runFakeJob(ctx, "cluster create")
	return nil
}

func (cluster *kubeProvider) Scale(ctx *api.Cluster, req *proto.ClusterReconfigureRequest) error {
	runFakeJob(ctx, "cluster scale")
	return nil
}

func (cluster *kubeProvider) Delete(ctx *api.Cluster, req *proto.ClusterDeleteRequest) error {
	runFakeJob(ctx, "cluster delete")
	return nil
}

func (cluster *kubeProvider) SetVersion(ctx *api.Cluster, req *proto.ClusterReconfigureRequest) error {
	runFakeJob(ctx, "cluster set version")
	return nil
}

func (cluster *kubeProvider) UploadStartupConfig(ctx *api.Cluster) error {
	return nil
}

func (cluster *kubeProvider) GetInstance(ctx *api.Cluster, md *api.InstanceMetadata) (*api.KubernetesInstance, error) {
	return &api.KubernetesInstance{}, nil
}

func (cluster *kubeProvider) MatchInstance(i *api.KubernetesInstance, md *api.InstanceMetadata) bool {
	return true
}

func runFakeJob(ctx *api.Cluster, requestType string) {
	ctx.Logger().Infof("starting %v job", requestType)
	for i := 1; i <= 10; i++ {
		ctx.Logger().Info(fmt.Sprint("Job completed: ", i*10, "%"))
		time.Sleep(time.Second * 3)
	}
}
