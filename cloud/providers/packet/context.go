package packet

import (
	"context"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	ctx      context.Context
	cluster  *api.Cluster
	conn     *cloudConnector
	actuator *ClusterActuator
	namer    namer
	m        sync.Mutex

	owner string
}

var _ Interface = &ClusterManager{}

const (
	UID      = "packet"
	Recorder = "packet-controller"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
	return &ClusterManager{ctx: ctx}
}

type paramK8sClient struct{}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	ma := NewMachineActuator(MachineActuatorParams{
		Ctx:           cm.ctx,
		EventRecorder: mgr.GetRecorder(Recorder),
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Owner:         cm.owner,
	})
	common.RegisterClusterProvisioner(UID, ma)
	return nil
}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	cm.m.Lock()
	defer cm.m.Unlock()

	v := cm.ctx.Value(paramK8sClient{})
	if kc, ok := v.(kubernetes.Interface); ok && kc != nil {
		return kc, nil
	}

	var err error
	cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return nil, err
	}
	kc, err := NewAdminClient(cm.ctx, cm.cluster)
	if err != nil {
		return nil, err
	}
	cm.ctx = context.WithValue(cm.ctx, paramK8sClient{}, kc)
	return kc, nil
}
