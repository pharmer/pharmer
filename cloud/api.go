package cloud

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
)

type ClusterManager interface {
	Create(req *proto.ClusterCreateRequest) error
	Scale(req *proto.ClusterReconfigureRequest) error
	Delete(req *proto.ClusterDeleteRequest) error
	SetVersion(req *proto.ClusterReconfigureRequest) error
	// UploadStartupConfig() error

	GetInstance(md *api.InstanceMetadata) (*api.Instance, error)
	MatchInstance(i *api.Instance, md *api.InstanceMetadata) bool
}
