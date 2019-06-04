package gce

import (
	"errors"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	defaultNetwork = "default"
)

type ClusterManager struct {
	*cloud.CloudManager

	conn  *cloudConnector
	namer namer
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "gce"
)

func init() {
	cloud.RegisterCloudManager(UID, func(cluster *api.Cluster, certs *cloud.PharmerCertificates) cloud.Interface {
		return New(cluster, certs)
	})
}

func New(cluster *api.Cluster, certs *cloud.PharmerCertificates) cloud.Interface {
	return &ClusterManager{
		CloudManager: &cloud.CloudManager{
			Cluster: cluster,
			Certs:   certs,
		},
		namer: namer{
			cluster: cluster,
		},
	}
}

func (cm *ClusterManager) GetConnector() cloud.ClusterApiProviderComponent {
	panic(1)
	return nil
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return errors.New("not implemented")
}

// TODO: Verify
func (cm *ClusterManager) CreateCCMCredential() error {
	//cloud.CreateCredentialSecret(cm.AdminClient, cm.Cluster)
	return nil
}
