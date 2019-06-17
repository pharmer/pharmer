package dokube

import (
	"github.com/pharmer/pharmer/store"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type DokubeNodeGroupManager struct {
	conn *cloudConnector
	ng   *clusterapi.MachineSet
}

func NewDokubeNodeGroupManager(conn *cloudConnector, ng *clusterapi.MachineSet) *DokubeNodeGroupManager {
	return &DokubeNodeGroupManager{conn: conn, ng: ng}
}

func (igm *DokubeNodeGroupManager) Apply() error {
	np, err := igm.conn.getNodePool(igm.ng)
	if err != nil {
		return err
	}
	if np == nil {
		if err = igm.conn.addNodePool(igm.ng); err != nil {
			return err
		}
	} else if *igm.ng.Spec.Replicas == 0 || igm.ng.DeletionTimestamp != nil {
		if err = igm.conn.deleteNodePool(igm.ng); err != nil {
			return err
		}
		err = store.StoreProvider.MachineSet(igm.conn.Cluster.Name).Delete(igm.ng.Name)
		if err != nil {
			return err
		}
	} else if err = igm.conn.adjustNodePool(igm.ng); err != nil {
		return err
	}

	igm.ng.Status.Replicas = *igm.ng.Spec.Replicas
	_, err = store.StoreProvider.MachineSet(igm.conn.Cluster.Name).UpdateStatus(igm.ng)
	if err != nil {
		return err
	}

	return nil
}
