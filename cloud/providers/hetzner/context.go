package hetzner

import (
	"context"
	"sync"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
	m       sync.Mutex
}

var _ Interface = &ClusterManager{}

const (
	UID = "hetzner"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) Scale(req *proto.ClusterReconfigureRequest) error {
	return UnsupportedOperation
}

func (cm *ClusterManager) SetVersion(req *proto.ClusterReconfigureRequest) error {
	return UnsupportedOperation
}

func (cm *ClusterManager) UploadStartupConfig() error {
	return UnsupportedOperation
}

func (cm *ClusterManager) GetInstance(md *api.NodeStatus) (*api.Node, error) {
	conn, err := NewConnector(cm.ctx, nil)
	if err != nil {
		return nil, err
	}
	im := &instanceManager{conn: conn}
	return im.GetInstance(md)
}

func (cm *ClusterManager) MatchInstance(i *api.Node, md *api.NodeStatus) bool {
	return i.Status.PrivateIP == md.PublicIP
}

type paramK8sClient struct{}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	cm.m.Lock()
	defer cm.m.Unlock()

	v := cm.ctx.Value(paramK8sClient{})
	if kc, ok := v.(kubernetes.Interface); ok && kc != nil {
		return kc, nil
	}

	kc, err := NewAdminClient(cm.ctx, cm.cluster)
	if err != nil {
		return nil, err
	}
	cm.ctx = context.WithValue(cm.ctx, paramK8sClient{}, kc)
	return kc, nil
}
