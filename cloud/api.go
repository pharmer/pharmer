package cloud

import (
	"context"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
)

type Cloud interface {
	Clusters() Provider
	Credentials() CloudCredential
}

type Provider interface {
	Create(ctx context.Context, req *proto.ClusterCreateRequest) error
	Scale(ctx context.Context, req *proto.ClusterReconfigureRequest) error
	Delete(ctx context.Context, req *proto.ClusterDeleteRequest) error
	SetVersion(ctx context.Context, req *proto.ClusterReconfigureRequest) error
	UploadStartupConfig(ctx context.Context) error

	GetInstance(ctx context.Context, md *api.InstanceMetadata) (*api.KubernetesInstance, error)
	MatchInstance(i *api.KubernetesInstance, md *api.InstanceMetadata) bool
}

type CloudCredential interface {
	IsValid() bool
	AsMap() map[string]string
}
