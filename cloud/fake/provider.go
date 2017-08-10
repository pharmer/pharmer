package fake

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/extpoints"
)

func init() {
	extpoints.KubeProviders.Register(new(kubeProvider), "fake")
}

type kubeProvider struct {
}

var _ extpoints.KubeProvider = &kubeProvider{}

func (cluster *kubeProvider) Create(ctx *contexts.ClusterContext, req *proto.ClusterCreateRequest) error {
	runFakeJob(ctx, "cluster create")
	return nil
}

func (cluster *kubeProvider) Scale(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) error {
	runFakeJob(ctx, "cluster scale")
	return nil
}

func (cluster *kubeProvider) Delete(ctx *contexts.ClusterContext, req *proto.ClusterDeleteRequest) error {
	runFakeJob(ctx, "cluster delete")
	return nil
}

func (cluster *kubeProvider) SetVersion(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) error {
	runFakeJob(ctx, "cluster set version")
	return nil
}

func (cluster *kubeProvider) UploadStartupConfig(ctx *contexts.ClusterContext) error {
	return nil
}

func (cluster *kubeProvider) GetInstance(ctx *contexts.ClusterContext, md *contexts.InstanceMetadata) (*contexts.KubernetesInstance, error) {
	return &contexts.KubernetesInstance{}, nil
}

func (cluster *kubeProvider) MatchInstance(i *contexts.KubernetesInstance, md *contexts.InstanceMetadata) bool {
	return true
}

func runFakeJob(ctx *contexts.ClusterContext, requestType string) {
	ctx.Notifier.Notify("only_notify", fmt.Sprintf("starting %v job", requestType))
	for i := 1; i <= 10; i++ {
		ctx.Logger().Info(fmt.Sprint("Job completed: ", i*10, "%"))
		time.Sleep(time.Second * 3)
	}
}
