package digitalocean

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"pharmer.dev/cloud/pkg/credential"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cloud/utils/kube"
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
	UID      = "digitalocean"
	Recorder = "digitalocean-controller"
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

func (cm *ClusterManager) SetCloudConnector() error {
	var err error

	if cm.conn, err = newconnector(cm); err != nil {
		cm.Logger.Error(err, "failed to get cloud connector")
		return err
	}

	return nil
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return ControllerManager, nil
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	log := cm.Logger
	cred, err := cm.GetCredential()
	if err != nil {
		log.Error(err, "failed to get credential for digitalocean")
		return err
	}

	err = kube.CreateSecret(kc, "digitalocean", metav1.NamespaceSystem, map[string][]byte{
		"access-token": []byte(cred.Spec.Data[credential.DigitalOceanToken]), //for ccm
		"token":        []byte(cred.Spec.Data[credential.DigitalOceanToken]), //for pharmer-flex and provisioner
	})
	if err != nil {
		log.Error(err, "failed to create ccm-secret")
		return err
	}

	return nil
}
