/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package packet

import (
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) EnsureMaster(masterMachine *v1alpha1.Machine) error {
	log := cm.Logger
	if d, _ := cm.conn.instanceIfExists(masterMachine); d == nil {
		log.Info("Creating master instance")
		var masterServer *api.NodeInfo
		nodeAddresses := make([]core.NodeAddress, 0)

		script, err := cloud.RenderStartupScript(cm, masterMachine, "", customTemplate)
		if err != nil {
			log.Error(err, "failed to render startup script")
			return err
		}

		masterServer, err = cm.conn.CreateInstance(masterMachine, script)
		if err != nil {
			log.Error(err, "failed to create instance")
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

		if err = cm.Cluster.SetClusterAPIEndpoints(nodeAddresses); err != nil {
			log.Error(err, "failed to set cluster api end points")
			return err
		}
		if _, err = cm.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
			log.Error(err, "failed to update cluster in store")
			return err
		}
	}

	return nil
}

func (cm *ClusterManager) PrepareCloud() error {
	log := cm.Logger
	log.Info("Preparing Cloud infra")

	found, _, err := cm.conn.getPublicKey()
	if err != nil {
		log.Error(err, "failed to get public key")
		return err
	}

	if !found {
		cm.Cluster.Status.Cloud.SSHKeyExternalID, err = cm.conn.importPublicKey()
		if err != nil {
			log.Error(err, "failed to import public key")
			return err
		}
	}

	return err
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	cm.Logger.Info("setting master sku", "sku", "baremetal_0")
	return "baremetal_0"
}

// Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) ApplyDelete() error {
	log := cm.Logger

	kc, err := cm.GetAdminClient()
	if err != nil {
		log.Error(err, "failed to get admin client")
		return err
	}

	masterInstances, err := kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.RoleMasterKey: "",
		}).String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		log.Error(err, "master instance not found")
	} else if err == nil {
		for _, mi := range masterInstances.Items {
			err = cm.conn.DeleteInstanceByProviderID(mi.Spec.ProviderID)
			if err != nil {
				log.Error(err, "failed to delete instance", "instance-id", mi.Spec.ProviderID)
			}
		}
	}

	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = cm.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster status in store")
		return err
	}

	log.Info("successfully deleted cluster")
	return nil
}
