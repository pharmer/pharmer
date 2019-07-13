package linode

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cloud/utils/kube"
)

type ClusterManager struct {
	*cloud.Scope

	conn  *cloudConnector
	namer namer
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID      = "linode"
	Recorder = "linode-controller"
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

func (cm *ClusterManager) ApplyScale() error {
	panic("implement me")
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	log := cm.Logger
	cred, err := cm.GetCredential()
	if err != nil {
		log.Error(err, "failed to get credential from store")
		return err
	}
	err = kube.CreateCredentialSecret(kc, cm.Cluster.CloudProvider(), metav1.NamespaceSystem, cred.Spec.Data)
	if err != nil {
		log.Error(err, "failed to create credential for pharmer-flex")
		return err
	}

	err = kube.CreateSecret(kc, "ccm-linode", metav1.NamespaceSystem, map[string][]byte{
		"apiToken": []byte(cred.Spec.Data["token"]),
		"region":   []byte(cm.Cluster.ClusterConfig().Cloud.Region),
	})
	if err != nil {
		log.Error(err, "failed to create ccm-secret")
		return err
	}
	return nil
}

func (cm *ClusterManager) SetCloudConnector() error {
	var err error

	if cm.conn, err = newconnector(cm); err != nil {
		cm.Logger.Error(err, "failed to get linode cloud connector")
		return err
	}

	return nil
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return ControllerManager, nil
}
