package aws

import (
	go_ctx "context"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/context"
)

const (
	UID = "aws"
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
	return (&clusterManager{ctx: context.NewContext(ctx, p.cfg)}).scale(req)
}

func (p *provider) Delete(ctx go_ctx.Context, req *proto.ClusterDeleteRequest) error {
	return (&clusterManager{ctx: context.NewContext(ctx, p.cfg)}).delete(req)
}

func (p *provider) SetVersion(ctx go_ctx.Context, req *proto.ClusterReconfigureRequest) error {
	return (&clusterManager{ctx: context.NewContext(ctx, p.cfg)}).setVersion(req)
}

func (p *provider) UploadStartupConfig(ctx go_ctx.Context) error {
	conn, err := NewConnector(nil)
	if err != nil {
		return err
	}
	cm := &clusterManager{ctx: context.NewContext(ctx, p.cfg), conn: conn}
	return cm.UploadStartupConfig()
}

func (p *provider) GetInstance(ctx go_ctx.Context, md *api.InstanceMetadata) (*api.KubernetesInstance, error) {
	conn, err := NewConnector(nil)
	if err != nil {
		return nil, err
	}
	cm := &clusterManager{ctx: context.NewContext(ctx, p.cfg), conn: conn}
	i, err := cm.newKubeInstance(md.ExternalID)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// TODO: Role not set
	return i, nil
}

func (p *provider) MatchInstance(i *api.KubernetesInstance, md *api.InstanceMetadata) bool {
	return i.ExternalID == md.ExternalID
}
