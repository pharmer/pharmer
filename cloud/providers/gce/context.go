package gce

import (
	"errors"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"k8s.io/client-go/kubernetes"
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

func (cm *ClusterManager) ApplyScale() error {
	panic("implement me")
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "gce"
)

func init() {
	cloud.RegisterCloudManager(UID, New)
}

func New(cluster *api.Cluster, certs *certificates.PharmerCertificates) cloud.Interface {
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

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	return errors.New("not implemented")
}

// TODO: Verify
func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	//cloud.CreateCredentialSecret(cm.AdminClient, cm.Cluster)
	return nil
}
