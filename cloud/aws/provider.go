package aws

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/extpoints"
)

func init() {
	extpoints.KubeProviders.Register(new(kubeProvider), "aws")
}

type kubeProvider struct {
}

var _ extpoints.KubeProvider = &kubeProvider{}

func (kp *kubeProvider) Create(ctx *contexts.ClusterContext, req *proto.ClusterCreateRequest) error {
	return (&clusterManager{ctx: ctx}).create(req)
}

func (kp *kubeProvider) Scale(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) error {
	return (&clusterManager{ctx: ctx}).scale(req)
}

func (kp *kubeProvider) Delete(ctx *contexts.ClusterContext, req *proto.ClusterDeleteRequest) error {
	return (&clusterManager{ctx: ctx}).delete(req)
}

func (kp *kubeProvider) SetVersion(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) error {
	return (&clusterManager{ctx: ctx}).setVersion(req)
}

func (cluster *kubeProvider) UploadStartupConfig(ctx *contexts.ClusterContext) error {
	conn, err := NewConnector(ctx)
	if err != nil {
		return err
	}
	cm := &clusterManager{ctx: ctx, conn: conn}
	return cm.UploadStartupConfig()
}

func (kp *kubeProvider) GetInstance(ctx *contexts.ClusterContext, md *contexts.InstanceMetadata) (*contexts.KubernetesInstance, error) {
	conn, err := NewConnector(ctx)
	if err != nil {
		return nil, err
	}
	cm := &clusterManager{ctx: ctx, conn: conn}
	i, err := cm.newKubeInstance(md.ExternalID)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(ctx).Err()
	}
	// TODO: Role not set
	return i, nil
}

func (kp *kubeProvider) MatchInstance(i *contexts.KubernetesInstance, md *contexts.InstanceMetadata) bool {
	return i.ExternalID == md.ExternalID
}
