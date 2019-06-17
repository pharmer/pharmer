package linode

import (
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/utils/certificates"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	UID      = "linode"
	Recorder = "linode-controller"
)

func init() {
	cloud.RegisterCloudManager(UID, New)
}

func New(cluster *api.Cluster, certs *certificates.Certificates) cloud.Interface {
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

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	err := kube.CreateCredentialSecret(kc, cm.Cluster, metav1.NamespaceSystem)
	if err != nil {
		return errors.Wrapf(err, "failed to create credential for pharmer-flex")
	}

	// create ccm secret
	cred, err := store.StoreProvider.Credentials().Get(cm.Cluster.Spec.Config.CredentialName)
	if err != nil {
		return err
	}

	err = kube.CreateSecret(kc, "ccm-linode", metav1.NamespaceSystem, map[string][]byte{
		"apiToken": []byte(cred.Spec.Data["token"]),
		"region":   []byte(cm.Cluster.ClusterConfig().Cloud.Region),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create ccm-secret")
	}
	return nil
}

func (cm *ClusterManager) GetCloudConnector() error {
	var err error

	if cm.conn, err = NewConnector(cm); err != nil {
		return err
	}

	return nil
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return ControllerManager, nil
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	ma := NewMachineActuator(MachineActuatorParams{
		EventRecorder: mgr.GetEventRecorderFor(Recorder),
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
	})
	common.RegisterClusterProvisioner(UID, ma)
	return nil
}
