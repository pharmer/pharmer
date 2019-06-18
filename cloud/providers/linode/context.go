package linode

import (
	"github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	cred, err := cm.GetCredential()
	if err != nil {
		return err
	}
	err = kube.CreateCredentialSecret(kc, cm.Cluster.CloudProvider(), metav1.NamespaceSystem, cred.Spec.Data)
	if err != nil {
		return errors.Wrapf(err, "failed to create credential for pharmer-flex")
	}

	// create ccm secret

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
