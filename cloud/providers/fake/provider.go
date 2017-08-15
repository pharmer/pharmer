package fake

import (
	go_ctx "context"
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/context"
)

const (
	UID = "fake"
)

func init() {
	cloud.RegisterProvider(UID, func(cfg *config.PharmerConfig) (cloud.Provider, error) { return &provider{cfg: cfg}, nil })
}

type provider struct {
	cfg *config.PharmerConfig
}

var _ cloud.Provider = &provider{}

func (p *provider) Create(ctx go_ctx.Context, req *proto.ClusterCreateRequest) error {
	p.runFakeJob(ctx, "cluster create")
	return nil
}

func (p *provider) Scale(ctx go_ctx.Context, req *proto.ClusterReconfigureRequest) error {
	p.runFakeJob(ctx, "cluster scale")
	return nil
}

func (p *provider) Delete(ctx go_ctx.Context, req *proto.ClusterDeleteRequest) error {
	p.runFakeJob(ctx, "cluster delete")
	return nil
}

func (p *provider) SetVersion(ctx go_ctx.Context, req *proto.ClusterReconfigureRequest) error {
	p.runFakeJob(ctx, "cluster set version")
	return nil
}

func (p *provider) UploadStartupConfig(ctx go_ctx.Context) error {
	return nil
}

func (p *provider) GetInstance(ctx go_ctx.Context, md *api.InstanceMetadata) (*api.KubernetesInstance, error) {
	return &api.KubernetesInstance{}, nil
}

func (p *provider) MatchInstance(i *api.KubernetesInstance, md *api.InstanceMetadata) bool {
	return true
}

func (p *provider) runFakeJob(ctx go_ctx.Context, requestType string) {
	c := context.NewContext(ctx, p.cfg)
	c.Logger().Infof("starting %v job", requestType)
	for i := 1; i <= 10; i++ {
		c.Logger().Info(fmt.Sprint("Job completed: ", i*10, "%"))
		time.Sleep(time.Second * 3)
	}
}
