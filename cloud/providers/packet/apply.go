package packet

import (
	"encoding/json"

	"github.com/appscode/go/log"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func (cm *ClusterManager) EnsureMaster() error {
	masterMachine, err := GetLeaderMachine(cm.Cluster)
	if err != nil {
		return err
	}
	if d, _ := cm.conn.instanceIfExists(masterMachine); d == nil {
		log.Info("Creating master instance")
		var masterServer *api.NodeInfo
		nodeAddresses := make([]core.NodeAddress, 0)

		script, err := RenderStartupScript(cm, masterMachine, "", customTemplate)
		if err != nil {
			return err
		}

		masterServer, err = cm.conn.CreateInstance(masterMachine, script)
		if err != nil {
			return err
		}
		if masterServer.PrivateIP != "" {
			nodeAddresses = append(nodeAddresses, core.NodeAddress{
				Type:    core.NodeInternalIP,
				Address: masterServer.PrivateIP,
			})
		}
		if masterServer.PublicIP != "" {
			nodeAddresses = append(nodeAddresses, core.NodeAddress{
				Type:    core.NodeExternalIP,
				Address: masterServer.PublicIP,
			})
		}

		if err = cm.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
			return err
		}
		if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
			return err
		}
	}

	return nil
}

func (cm *ClusterManager) PrepareCloud() error {
	found, _, err := cm.conn.getPublicKey()
	if err != nil {
		return err
	}

	if !found {
		cm.Cluster.Status.Cloud.SShKeyExternalID, err = cm.conn.importPublicKey()
		if err != nil {
			return err
		}
	}

	return err
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	return "baremetal_0"
}

// Creates network, and creates ready master(s)
func (cm *ClusterManager) applyCreate() error {

	return nil
}

// createSecrets creates all the secrets necessary for creating a cluster
// it creates credential for ccm, pharmer-flex, pharmer-provisioner
func (cm *ClusterManager) createSecrets(kc kubernetes.Interface) error {
	// pharmer-flex secret
	if err := CreateCredentialSecret(kc, cm.Cluster, ""); err != nil {
		return errors.Wrapf(err, "failed to create flex-secret")
	}

	// ccm-secret
	cred, err := store.StoreProvider.Credentials().Get(cm.Cluster.ClusterConfig().CredentialName)
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster cred")
	}
	typed := credential.Packet{CommonSpec: credential.CommonSpec(cred.Spec)}
	ok, err := typed.IsValid()
	if !ok {
		return errors.New("credential not valid")
	}
	cloudConfig := &api.PacketCloudConfig{
		Project: typed.ProjectID(),
		ApiKey:  typed.APIKey(),
		Zone:    cm.Cluster.ClusterConfig().Cloud.Zone,
	}
	data, err := json.Marshal(cloudConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal cloud-config")
	}
	err = CreateSecret(kc, "cloud-config", metav1.NamespaceSystem, map[string][]byte{
		"cloud-config": data,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create cloud-config")
	}
	return nil
}

// Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) ApplyDelete() error {
	log.Infoln("deleting cluster")

	if cm.Cluster.Status.Phase == api.ClusterReady {
		cm.Cluster.Status.Phase = api.ClusterDeleting
	}
	_, err := store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}

	err = DeleteAllWorkerMachines(cm)
	if err != nil {
		log.Infof("failed to delete nodes: %v", err)
	}

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return err
	}

	var masterInstances *core.NodeList
	masterInstances, err = kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.RoleMasterKey: "",
		}).String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		log.Infof("master instance not found. Reason: %v", err)
	} else if err == nil {
		for _, mi := range masterInstances.Items {
			err = cm.conn.DeleteInstanceByProviderID(mi.Spec.ProviderID)
			if err != nil {
				log.Infof("Failed to delete instance %s. Reason: %s", mi.Spec.ProviderID, err)
			}
		}
	}

	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}

	log.Infof("Cluster %v deletion is deleted successfully", cm.Cluster.Name)
	return nil
}
