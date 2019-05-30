package gce

import (
	"errors"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	defaultNetwork = "default"
)

type ClusterManager struct {
	cluster     *api.Cluster
	certs       *api.PharmerCertificates
	conn        *cloudConnector
	namer       namer
	adminClient kubernetes.Interface
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
		namer: namer{
			cluster: cluster,
		},
	}
}

func (cm *ClusterManager) SetAdminClient(kc kubernetes.Interface) {
	cm.adminClient = kc
}

func (cm *ClusterManager) GetCluster() *api.Cluster {
	return cm.cluster
}

func (cm *ClusterManager) GetAdminClient() kubernetes.Interface {
	return cm.adminClient
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
