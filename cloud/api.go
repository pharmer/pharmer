package cloud

import (
	"errors"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
)

var InstanceNotFound = errors.New("Instance not found")
var UnsupportedOperation = errors.New("Unsupported operation")

type ClusterManager interface {
	Create(req *proto.ClusterCreateRequest) error
	Scale(req *proto.ClusterReconfigureRequest) error
	Delete(req *proto.ClusterDeleteRequest) error
	SetVersion(req *proto.ClusterReconfigureRequest) error

	GetInstance(md *api.InstanceStatus) (*api.Instance, error)
}
