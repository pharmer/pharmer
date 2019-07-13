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
