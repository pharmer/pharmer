package azure

import (
	"context"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
	m       sync.Mutex

	owner string
}

var _ Interface = &ClusterManager{}

const (
	UID = "azure"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
	return &ClusterManager{ctx: ctx}
}

// AddToManager adds all Controllers to the Manager
func (cm *ClusterManager) AddToManager(ctx context.Context, m manager.Manager) error {
	return nil
}

type paramK8sClient struct{}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return nil
}

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
