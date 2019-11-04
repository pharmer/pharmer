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
package digitalocean

import (
	"context"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"

	core "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) EnsureMaster(leaderMachine *v1alpha1.Machine) error {
	log := cm.Logger.WithValues("machine-name", leaderMachine.Name)
	log.Info("ensuring master machine")

	if d, _ := cm.conn.instanceIfExists(leaderMachine); d == nil {
		log.Info("Creating master instance")
		nodeAddresses := make([]core.NodeAddress, 0)

		if cm.Cluster.Status.Cloud.LoadBalancer.DNS != "" {
			nodeAddresses = append(nodeAddresses, core.NodeAddress{
				Type:    core.NodeExternalDNS,
				Address: cm.Cluster.Status.Cloud.LoadBalancer.DNS,
			})
		} else if cm.Cluster.Status.Cloud.LoadBalancer.IP != "" {
			nodeAddresses = append(nodeAddresses, core.NodeAddress{
				Type:    core.NodeExternalIP,
				Address: cm.Cluster.Status.Cloud.LoadBalancer.IP,
			})
		}

		script, err := cloud.RenderStartupScript(cm, leaderMachine, "", customTemplate)
		if err != nil {
			log.Error(err, "failed to render start up script")
			return err
		}

		err = cm.conn.CreateInstance(cm.Cluster, leaderMachine, script)
		if err != nil {
			log.Error(err, "failed to create instance")
			return err
		}

		if err = cm.Cluster.SetClusterAPIEndpoints(nodeAddresses); err != nil {
			log.Error(err, "failed to set cluster api endpoints")
			return err
		}
	}
	log.Info("successfully created cluster")

	var err error
	cm.Cluster, err = cm.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster in store")
		return err
	}

	return nil
}

func (cm *ClusterManager) PrepareCloud() error {
	log := cm.Logger
	log.Info("preparing cloud infra")

	var found bool
	var err error

	found, _, err = cm.conn.getPublicKey()
	if err != nil {
		log.Error(err, "failed to get public key")
		return err
	}
	if !found {
		log.V(2).Info("public key not found, importing")
		_, err = cm.conn.importPublicKey()
		if err != nil {
			log.Error(err, "failed to import public key")
			return err
		}
	}

	// ignore errors, since tags are simply informational.
	found, err = cm.conn.getTags()
	if err != nil {
		log.Error(err, "failed to get tags")
		return err
	}
	if !found {
		if err = cm.conn.createTags(); err != nil {
			log.Error(err, "failed to create tags")
			return err
		}
	}

	lb, err := cm.conn.lbByName(context.Background(), cm.namer.LoadBalancerName())
	if err == errLBNotFound {
		lb, err = cm.conn.createLoadBalancer(context.Background(), cm.namer.LoadBalancerName())
		if err != nil {
			log.Error(err, "failed to create loadbalancer")
			return err
		}
	}

	cm.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   lb.IP,
		Port: lb.ForwardingRules[0].EntryPort,
	}

	nodeAddresses := []corev1.NodeAddress{
		{
			Type:    corev1.NodeExternalIP,
			Address: cm.Cluster.Status.Cloud.LoadBalancer.IP,
		},
	}

	if err = cm.Cluster.SetClusterAPIEndpoints(nodeAddresses); err != nil {
		log.Error(err, "failed to set control plane endpoints")
		return err
	}

	log.Info("successfully created cloud infra")
	return nil
}

// Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) ApplyDelete() error {
	log := cm.Logger

	kc, err := cm.GetAdminClient()
	if err != nil {
		log.Error(err, "failed to get admin client")
		return err
	}
	var masterInstances *core.NodeList
	masterInstances, err = kc.CoreV1().Nodes().List(metav1.ListOptions{
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
				log.Error(err, "Failed to delete instance", "instanceID", mi.Spec.ProviderID)
			}
		}
	}

	// delete by tag
	tag := "KubernetesCluster:" + cm.Cluster.Name
	_, err = cm.conn.client.Droplets.DeleteByTag(context.Background(), tag)
	if err != nil {
		log.Error(err, "Failed to delete resources", "tag", tag)
	}
	log.Info("Deleted droplet", "tag", tag)

	// Delete SSH key
	found, _, err := cm.conn.getPublicKey()
	if err != nil {
		log.Error(err, "failed to get public key")
		return err
	}
	if found {
		err = cm.conn.deleteSSHKey()
		if err != nil {
			log.Error(err, "failed to delete ssh key")
			return err
		}
	}

	_, err = cm.conn.lbByName(context.Background(), cm.namer.LoadBalancerName())
	if err != errLBNotFound {
		if err = cm.conn.deleteLoadBalancer(context.Background(), cm.namer.LoadBalancerName()); err != nil {
			log.Error(err, "failed to delete load balancer")
			return err
		}

	}

	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = cm.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster status in store")
		return err
	}

	log.Info("successfully deleted cluster")
	return err
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	cm.Logger.Info("setting master sku", "sku", "2gb")
	return "2gb"
}
