package azure

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	UID = "azure"
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

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	panic("implement me")
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	cred, err := cm.GetCredential()
	if err != nil {
		return err
	}

	if err := kube.CreateNamespace(kc, "azure-provider-system"); err != nil {
		return err
	}

	data := cred.Spec.Data
	if err := kube.CreateSecret(kc, "azure-provider-azure-controller-secrets", "azure-provider-system", map[string][]byte{
		"client-id":       []byte(data["clientID"]),
		"client-secret":   []byte(data["clientSecret"]),
		"subscription-id": []byte(data["subscriptionID"]),
		"tenant-id":       []byte(data["tenantID"]),
	}); err != nil {
		return err
	}
	return nil
}

func (cm *ClusterManager) SetCloudConnector() error {
	var err error
	cm.conn, err = newconnector(cm)
	return err
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return ClusterAPIComponents, nil
}

func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	panic("implement me")
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	return kube.GetAdminConfig(cm.Cluster, cm.GetCaCertPair())
}
