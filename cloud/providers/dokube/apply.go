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
package dokube

import (
	"context"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud/utils/certificates"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) PrepareCloud() error {
	log := cm.Logger
	if cm.Cluster.Spec.Config.Cloud.Dokube.ClusterID == "" {
		cluster, err := cm.conn.createCluster(cm.Cluster)
		if err != nil {
			log.Error(err, "failed to create cluster")
			return err
		}

		cm.Cluster.Spec.Config.Cloud.Dokube.ClusterID = cluster.ID

		if err := cm.retrieveClusterStatus(cluster); err != nil {
			log.Error(err, "failed to retrieve cluster status")
			return err
		}

		err = cm.StoreCertificate(cm.conn.client)
		if err != nil {
			log.Error(err, "failed to store certs in store")
			return err
		}
		certs, err := certificates.GetPharmerCerts(cm.StoreProvider, cm.Cluster.Name)
		if err != nil {
			log.Error(err, "failed to get certs")
			return err
		}

		cm.Certs = certs

		if _, err = cm.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
			log.Error(err, "failed to update cluster in store")
			return err
		}
	}

	return nil
}

func (cm *ClusterManager) ApplyScale() error {
	log := cm.Logger

	var nodeGroups []*clusterapi.MachineSet
	nodeGroups, err := cm.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list machinesets from store")
		return err
	}
	for _, ng := range nodeGroups {
		igm := NewDokubeNodeGroupManager(cm.Scope, cm.conn, ng)

		err = igm.Apply()
		if err != nil {
			log.Error(err, "failed to apply node groups")
			return err
		}
	}
	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluser in store")
		return err
	}
	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	log := cm.Logger
	if cm.Cluster.Status.Phase == api.ClusterReady {
		cm.Cluster.Status.Phase = api.ClusterDeleting
	}
	_, err := cm.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster status in store")
		return err
	}
	_, err = cm.conn.client.Kubernetes.Delete(context.Background(), cm.conn.Cluster.Spec.Config.Cloud.Dokube.ClusterID)
	if err != nil {
		log.Error(err, "failed to delete digitalocean cluster")
		return err
	}
	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster")
		return err
	}

	return nil
}
