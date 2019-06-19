package packet

import (
	"encoding/json"

	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
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
	UID      = "packet"
	Recorder = "packet-controller"
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
		return errors.Wrapf(err, "failed to get cluster cred")
	}
	// pharmer-flex secret
	if err := kube.CreateCredentialSecret(kc, cm.Cluster.CloudProvider(), metav1.NamespaceSystem, cred.Spec.Data); err != nil {
		return errors.Wrapf(err, "failed to create flex-secret")
	}

	// ccm-secret
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	ok, err := typed.IsValid()
	if !ok {
		return errors.New("credential not valid")
	}
	if err != nil {
		return err
	}
	cloudConfig := &api.PacketCloudConfig{
		Project: typed.ProjectID(),
		APIKey:  typed.APIKey(),
		Zone:    cm.Cluster.ClusterConfig().Cloud.Zone,
	}
	data, err := json.Marshal(cloudConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal cloud-config")
	}
	err = kube.CreateSecret(kc, "cloud-config", metav1.NamespaceSystem, map[string][]byte{
		"cloud-config": data,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create cloud-config")
	}
	return nil
}

func (cm *ClusterManager) GetCloudConnector() error {
	var err error

	if cm.conn, err = newconnector(cm); err != nil {
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
