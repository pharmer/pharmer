package packet

import (
	"encoding/json"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1alpha1"
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
	log := cm.Logger
	cred, err := cm.GetCredential()
	if err != nil {
		log.Error(err, "failed to get cluster credential from store")
		return err
	}
	// pharmer-flex secret
	if err := kube.CreateCredentialSecret(kc, cm.Cluster.CloudProvider(), metav1.NamespaceSystem, cred.Spec.Data); err != nil {
		log.Error(err, "failed to creat4e flex-secret")
		return err
	}

	// ccm-secret
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	ok, err := typed.IsValid()
	if !ok {
		return errors.New("credential not valid")
	}
	if err != nil {
		log.Error(err, "credential is not valid")
		return err
	}
	cloudConfig := &api.PacketCloudConfig{
		Project: typed.ProjectID(),
		APIKey:  typed.APIKey(),
		Zone:    cm.Cluster.ClusterConfig().Cloud.Zone,
	}
	data, err := json.Marshal(cloudConfig)
	if err != nil {
		log.Error(err, "failed to json masrshal cloud config")
		return err
	}
	err = kube.CreateSecret(kc, "cloud-config", metav1.NamespaceSystem, map[string][]byte{
		"cloud-config": data,
	})
	if err != nil {
		log.Error(err, "failed to create cloud config")
		return errors.Wrapf(err, "failed to create cloud-config")
	}
	return nil
}

func (cm *ClusterManager) SetCloudConnector() error {
	var err error

	if cm.conn, err = newconnector(cm); err != nil {
		cm.Logger.Error(err, "failed to set packet cloud connector")
		return err
	}

	return nil
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return ControllerManager, nil
}
