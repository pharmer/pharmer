package hetzner

import (
	go_ctx "context"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/context"
)

const (
	UID = "hetzner"
)

func init() {
	cloud.RegisterProvider(UID, func(cfg *config.PharmerConfig) (cloud.Provider, error) { return &provider{cfg: cfg}, nil })
}

type provider struct {
	cfg *config.PharmerConfig
}

var _ cloud.Provider = &provider{}

func (p *provider) Create(ctx go_ctx.Context, req *proto.ClusterCreateRequest) error {
	return (&clusterManager{ctx: context.NewContext(ctx, p.cfg)}).create(req)
}

func (p *provider) Scale(ctx go_ctx.Context, req *proto.ClusterReconfigureRequest) error {
	return cloud.UnsupportedOperation
}

func (p *provider) Delete(ctx go_ctx.Context, req *proto.ClusterDeleteRequest) error {
	return (&clusterManager{ctx: context.NewContext(ctx, p.cfg)}).delete(req)
}

func (p *provider) SetVersion(ctx go_ctx.Context, req *proto.ClusterReconfigureRequest) error {
	return cloud.UnsupportedOperation
}

func (p *provider) UploadStartupConfig(ctx go_ctx.Context) error {
	return cloud.UnsupportedOperation
	/*conn, err := NewConnector(ctx)
	if err != nil {
		return err
	}
	cm := &clusterManager{ctx: ctx, conn: conn}
	return cm.UploadStartupConfig()*/
}

func (p *provider) GetInstance(ctx go_ctx.Context, md *api.InstanceMetadata) (*api.Instance, error) {
	c := context.NewContext(ctx, p.cfg)
	conn, err := NewConnector(c, nil)
	if err != nil {
		return nil, err
	}
	im := &instanceManager{conn: conn}
	return im.GetInstance(md)
}

func (p *provider) MatchInstance(i *api.Instance, md *api.InstanceMetadata) bool {
	return i.Status.InternalIP == md.ExternalIP
}
