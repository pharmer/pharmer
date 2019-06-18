package packet

import (
	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) EnsureMaster(masterMachine *v1alpha1.Machine) error {
	if d, _ := cm.conn.instanceIfExists(masterMachine); d == nil {
		log.Info("Creating master instance")
		var masterServer *api.NodeInfo
		nodeAddresses := make([]core.NodeAddress, 0)

		script, err := cloud.RenderStartupScript(cm, masterMachine, "", customTemplate)
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
		if _, err = cm.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
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

// Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) ApplyDelete() error {
	kc, err := cm.GetAdminClient()
	if err != nil {
		return err
	}

	masterInstances, err := kc.CoreV1().Nodes().List(metav1.ListOptions{
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
	_, err = cm.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}

	log.Infof("Cluster %v deletion is deleted successfully", cm.Cluster.Name)
	return nil
}
