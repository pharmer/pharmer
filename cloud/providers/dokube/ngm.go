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
	"pharmer.dev/pharmer/cloud"

	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type DokubeNodeGroupManager struct {
	*cloud.Scope
	conn *cloudConnector
	ng   *clusterapi.MachineSet
}

func NewDokubeNodeGroupManager(scope *cloud.Scope, conn *cloudConnector, ng *clusterapi.MachineSet) *DokubeNodeGroupManager {
	return &DokubeNodeGroupManager{Scope: scope, conn: conn, ng: ng}
}

func (igm *DokubeNodeGroupManager) Apply() error {
	log := igm.Logger.WithValues("nodepool-name", igm.ng.Name)
	np, err := igm.conn.getNodePool(igm.ng)
	if err != nil {
		log.Error(err, "failed to get nodepool")
		return err
	}
	if np == nil {
		if err = igm.conn.addNodePool(igm.ng); err != nil {
			log.Error(err, "failed to add nodepool")
			return err
		}
	} else if *igm.ng.Spec.Replicas == 0 || igm.ng.DeletionTimestamp != nil {
		if err = igm.conn.deleteNodePool(igm.ng); err != nil {
			log.Error(err, "failed to delete nodepool")
			return err
		}
		err = igm.StoreProvider.MachineSet(igm.conn.Cluster.Name).Delete(igm.ng.Name)
		if err != nil {
			log.Error(err, "failed to delete machineset from store")
			return err
		}
	} else if err = igm.conn.adjustNodePool(igm.ng); err != nil {
		log.Error(err, "failed to adjust nodepool")
		return err
	}

	igm.ng.Status.Replicas = *igm.ng.Spec.Replicas
	_, err = igm.StoreProvider.MachineSet(igm.conn.Cluster.Name).UpdateStatus(igm.ng)
	if err != nil {
		log.Error(err, "failed to update cluster status in store")
		return err
	}

	return nil
}
