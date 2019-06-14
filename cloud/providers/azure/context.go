package azure

import (
	"github.com/pharmer/pharmer/store"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	*cloud.CloudManager

	conn  *cloudConnector
	namer namer
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "azure"
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

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	panic("implement me")
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	cred, err := store.StoreProvider.Credentials().Get(cm.Cluster.Spec.Config.CredentialName)
	if err != nil {
		return err
	}

	if err := cloud.CreateNamespace(kc, "azure-provider-system"); err != nil {
		return err
	}

	data := cred.Spec.Data
	if err := cloud.CreateSecret(kc, "azure-provider-azure-controller-secrets", "azure-provider-system", map[string][]byte{
		"client-id":       []byte(data["clientID"]),
		"client-secret":   []byte(data["clientSecret"]),
		"subscription-id": []byte(data["subscriptionID"]),
		"tenant-id":       []byte(data["tenantID"]),
	}); err != nil {
		return err
	}
	return nil
}

func (cm *ClusterManager) GetCloudConnector() error {
	panic("implement me")
}

func (cm *ClusterManager) NewNodeTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	panic("implement me")
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return ClusterAPIComponents, nil
}

func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	panic("implement me")
}

func (cm *ClusterManager) GetKubeConfig(cluster *api.Cluster) (*api.KubeConfig, error) {
	panic("implement me")
}
