package digitalocean

import (
	"github.com/pharmer/cloud/pkg/credential"
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
	UID      = "digitalocean"
	Recorder = "digitalocean-controller"
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
	ma := NewMachineActuator(MachineActuatorParams{
		EventRecorder: mgr.GetEventRecorderFor(Recorder),
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
	})
	// TODO: is this required?
	common.RegisterClusterProvisioner(UID, ma)
	return nil
}

func (cm *ClusterManager) GetCloudConnector() error {
	var err error

	if cm.conn, err = newConnector(cm); err != nil {
		return err
	}

	return nil
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return ControllerManager, nil
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	cred, err := store.StoreProvider.Credentials().Get(cm.Cluster.Spec.Config.CredentialName)
	if err != nil {
		return errors.Wrapf(err, "failed to get credential for digitalocean")
	}

	err = kube.CreateSecret(kc, "digitalocean", metav1.NamespaceSystem, map[string][]byte{
		"access-token": []byte(cred.Spec.Data[credential.DigitalOceanToken]), //for ccm
		"token":        []byte(cred.Spec.Data[credential.DigitalOceanToken]), //for pharmer-flex and provisioner
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create ccm-secret for digitalocean")
	}

	return nil
}

//func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
//
//	v := cm.ctx.Value(paramK8sClient{})
//	if kc, ok := v.(kubernetes.Interface); ok && kc != nil {
//		return kc, nil
//	}
//	var err error
//
//	//cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner)
//	//if err != nil {
//	//	return nil, err
//	//}
//	kc, err := NewAdminClient(cm.ctx, cm.cluster)
//	if err != nil {
//		return nil, err
//	}
//	cm.ctx = context.WithValue(cm.ctx, paramK8sClient{}, kc)
//	return kc, nil
//}
