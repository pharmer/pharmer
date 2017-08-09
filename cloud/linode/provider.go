package linode

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/cloud/extpoints"
	"github.com/appscode/pharmer/common"
	"github.com/appscode/pharmer/contexts"
)

func init() {
	extpoints.KubeProviders.Register(new(kubeProvider), "linode")
}

type kubeProvider struct {
}

var _ extpoints.KubeProvider = &kubeProvider{}

func (kp *kubeProvider) Create(ctx *contexts.ClusterContext, req *proto.ClusterCreateRequest) error {
	return (&clusterManager{ctx: ctx}).create(req)
}

func (kp *kubeProvider) Scale(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) error {
	return common.UnsupportedOperation
}

func (kp *kubeProvider) Delete(ctx *contexts.ClusterContext, req *proto.ClusterDeleteRequest) error {
	return (&clusterManager{ctx: ctx}).delete(req)
}

func (kp *kubeProvider) SetVersion(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) error {
	return common.UnsupportedOperation
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
	im := &instanceManager{conn: conn}
	return im.GetInstance(md)
}

func (kp *kubeProvider) MatchInstance(i *contexts.KubernetesInstance, md *contexts.InstanceMetadata) bool {
	return i.InternalIP == md.InternalIP
}
