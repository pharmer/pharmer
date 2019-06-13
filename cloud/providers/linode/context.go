package linode

import (
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	*CloudManager

	conn  *cloudConnector
	namer namer
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	// create secret for pharmer-flex and provisioner
	err := CreateCredentialSecret(kc, cm.Cluster, cm.Cluster.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create credential for pharmer-flex")
	}

	// create ccm secret
	cred, err := store.StoreProvider.Credentials().Get(cm.Cluster.Spec.Config.CredentialName)
	if err != nil {
		return err
	}

	err = CreateSecret(kc, "ccm-linode", metav1.NamespaceSystem, map[string][]byte{
		"apiToken": []byte(cred.Spec.Data["token"]),
		"region":   []byte(cm.Cluster.ClusterConfig().Cloud.Region),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create ccm-secret")
	}
	return nil
}

func (cm *ClusterManager) GetConnector() ClusterApiProviderComponent {
	panic(1)
	return nil
}

func (cm *ClusterManager) GetCloudConnector() error {
	var err error

	if cm.conn, err = NewConnector(cm); err != nil {
		return err
	}

	return nil
}

func (cm *ClusterManager) NewNodeTemplateData(machine *v1alpha1.Machine, token string, td TemplateData) TemplateData {
	panic("implement me")
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	panic("implement me")
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	panic("implement me")
}

var _ Interface = &ClusterManager{}

const (
	UID      = "linode"
	Recorder = "linode-controller"
)

func init() {
	RegisterCloudManager(UID, func(cluster *api.Cluster, certs *PharmerCertificates) Interface {
		return New(cluster, certs)
	})
}

func New(cluster *api.Cluster, certs *PharmerCertificates) Interface {
	return &ClusterManager{}
}

func (cm *ClusterManager) InitializeMachineActuator(mgr manager.Manager) error {
	ma := NewMachineActuator(MachineActuatorParams{
		EventRecorder: mgr.GetEventRecorderFor(Recorder),
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		//Owner:         cm.owner,
	})
	common.RegisterClusterProvisioner(UID, ma)
	return nil
}

/*func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	v := cm.ctx.Value(paramK8sClient{})
	if kc, ok := v.(kubernetes.Interface); ok && kc != nil {
		return kc, nil
	}
	var err error

	//cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner)
	//if err != nil {
	//	return nil, err
	//}
	kc, err := NewAdminClient(cm.ctx, cm.cluster)
	if err != nil {
		return nil, err
	}
	cm.ctx = context.WithValue(cm.ctx, paramK8sClient{}, kc)
	return kc, nil
}*/
