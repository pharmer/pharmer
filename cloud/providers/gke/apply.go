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
package gke

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) PrepareCloud() error {
	log := cm.Logger

	err := cm.SetCloudConnector()
	if err != nil {
		log.Error(err, "failed to set cloud connector")
		return err
	}

	found, _ := cm.conn.getNetworks()
	if !found {
		if err := cm.conn.ensureNetworks(); err != nil {
			log.Error(err, "failed to ensure networks")
			return err
		}
	}

	config := cm.Cluster.Spec.Config
	cluster, _ := cm.conn.containerService.Projects.Zones.Clusters.Get(config.Cloud.Project, config.Cloud.Zone, cm.Cluster.Name).Do()
	if cluster == nil {
		cluster, err = encodeCluster(cm.StoreProvider.MachineSet(cm.Cluster.Name), cm.Cluster)
		if err != nil {
			log.Error(err, "failed to encode cluster")
			return err
		}

		var op string
		if op, err = cm.conn.createCluster(cluster); err != nil {
			log.Error(err, "failed to create cluster")
			return err
		}
		if err = cm.conn.waitForZoneOperation(op); err != nil {
			log.Error(err, "zonal operation failed")
			return err
		}

		cluster, err = cm.conn.containerService.Projects.Zones.Clusters.Get(config.Cloud.Project, config.Cloud.Zone, cm.Cluster.Name).Do()
		if err != nil {
			log.Error(err, "failed to get cluster")
			return err
		}
		err = cm.StoreCertificate(cm.StoreProvider.Certificates(cm.Cluster.Name), cluster)
		if err != nil {
			log.Error(err, "failed to store certificate in store")
			return err
		}
	}
	if err = cm.retrieveClusterStatus(cluster); err != nil {
		log.Error(err, "failed to retrieve cluster status")
		return err
	}

	return nil
}

func (cm *ClusterManager) ApplyScale() error {
	log := cm.Logger

	nodeGroups, err := cm.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list nodegroups from store")
		return err
	}
	for _, ng := range nodeGroups {
		igm := NewGKENodeGroupManager(cm.Scope, cm.conn, ng)
		err = igm.Apply()
		if err != nil {
			log.Error(err, "failed to apply node group")
			return err
		}
	}

	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster in store")
		return err
	}
	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	log := cm.Logger

	_, err := cm.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster status")
		return err
	}
	var op string
	op, err = cm.conn.deleteCluster()
	if err != nil {
		log.Error(err, "failed to delete cluster")
		return err
	}
	if err = cm.conn.waitForZoneOperation(op); err != nil {
		log.Error(err, "zonal operation failed")
		cm.Cluster.Status.Reason = err.Error()
		return err
	}

	return nil
}
