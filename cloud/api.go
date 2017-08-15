package cloud

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
)

type Provider interface {
	Create(ctx *api.Cluster, req *proto.ClusterCreateRequest) error
	Scale(ctx *api.Cluster, req *proto.ClusterReconfigureRequest) error
	Delete(ctx *api.Cluster, req *proto.ClusterDeleteRequest) error
	SetVersion(ctx *api.Cluster, req *proto.ClusterReconfigureRequest) error
	UploadStartupConfig(ctx *api.Cluster) error

	GetInstance(ctx *api.Cluster, md *api.InstanceMetadata) (*api.KubernetesInstance, error)
	MatchInstance(i *api.KubernetesInstance, md *api.InstanceMetadata) bool
}
