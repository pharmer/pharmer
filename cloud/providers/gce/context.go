package gce

import (
	"github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultNetwork = "default"
)

type ClusterManager struct {
	*cloud.Scope

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

func New(s *cloud.Scope) cloud.Interface {
	return &ClusterManager{
		Scope: s,
		namer: namer{
			cluster: s.Cluster,
		},
	}
}

// TODO: Verify
func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	//cloud.CreateCredentialSecret(cm.AdminClient, cm.Cluster)
	return nil
}
