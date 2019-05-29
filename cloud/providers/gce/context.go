package gce

import (
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

	//ctx         context.Context
	conn        *cloudConnector
	namer       namer
	m           sync.Mutex
	adminClient kubernetes.Interface

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

func (cm *ClusterManager) GetCluster() *api.Cluster {
	return cm.cluster
}

func (cm *ClusterManager) GetAdminClient() kubernetes.Interface {
	return cm.adminClient
}

func (cm *ClusterManager) GetMutex() *sync.Mutex {
	return &cm.m
}

func (cm *ClusterManager) GetCaCertPair() *api.CertKeyPair {
	return &cm.certs.CACert
}

func (cm *ClusterManager) GetPharmerCertificates() *api.PharmerCertificates {
	return cm.certs
}

func (cm *ClusterManager) GetConnector() ClusterApiProviderComponent {
	return cm.conn
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return errors.New("not implemented")
}
