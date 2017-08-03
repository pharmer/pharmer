package extpoints

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/contexts"
)

type KubeProvider interface {
	Create(ctx *contexts.ClusterContext, req *proto.ClusterCreateRequest) error
	Scale(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) error
	Delete(ctx *contexts.ClusterContext, req *proto.ClusterDeleteRequest) error
	SetVersion(ctx *contexts.ClusterContext, req *proto.ClusterReconfigureRequest) error
	UploadStartupConfig(ctx *contexts.ClusterContext) error

	GetInstance(ctx *contexts.ClusterContext, md *contexts.InstanceMetadata) (*contexts.KubernetesInstance, error)
	MatchInstance(i *contexts.KubernetesInstance, md *contexts.InstanceMetadata) bool
}
