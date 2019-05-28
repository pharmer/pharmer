package gce

import (
	"context"
	"errors"
	"sync"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	maxInstancesPerMIG = 5 // Should be 500
	defaultNetwork     = "default"
)

type ClusterManager struct {
	cluster *api.Cluster
	certs   *api.PharmerCertificates

	ctx   context.Context
	conn  *cloudConnector
	namer namer
	m     sync.Mutex

	owner string
}

var _ Interface = &ClusterManager{}

const (
	UID = "gce"
)

func init() {
	RegisterCloudManager(UID, func(cluster *api.Cluster, certs *api.PharmerCertificates) Interface {
		return New(cluster, certs)
	})
}

func New(cluster *api.Cluster, certs *api.PharmerCertificates) Interface {
	return &ClusterManager{
		cluster: cluster,
		certs:   certs,
	}
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

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return errors.New("not implemented")
}
