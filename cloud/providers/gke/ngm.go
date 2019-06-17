package gke

import (
	"github.com/pharmer/pharmer/store"
	"google.golang.org/api/container/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type GKENodeGroupManager struct {
	conn *cloudConnector
	ng   *clusterapi.MachineSet
}

func NewGKENodeGroupManager(conn *cloudConnector, ng *clusterapi.MachineSet) *GKENodeGroupManager {
	return &GKENodeGroupManager{conn: conn, ng: ng}
}

func (igm *GKENodeGroupManager) Apply() error {
	var np *container.NodePool
	np, _ = igm.conn.containerService.Projects.Zones.Clusters.NodePools.Get(igm.conn.Cluster.Spec.Config.Cloud.Project, igm.conn.Cluster.Spec.Config.Cloud.Zone, igm.conn.Cluster.Name, igm.ng.Name).Do()

	if np == nil {
		op, err := igm.conn.addNodePool(igm.ng)
		if err != nil {
			return err
		}
		if err := igm.conn.waitForZoneOperation(op); err != nil {
			return err
		}

	} else if *igm.ng.Spec.Replicas == 0 || igm.ng.DeletionTimestamp != nil {
		op, err := igm.conn.deleteNoodPool(igm.ng)
		if err != nil {
			return err
		}
		if err = igm.conn.waitForZoneOperation(op); err != nil {
			return err
		}
		err = store.StoreProvider.MachineSet(igm.conn.Cluster.Name).Delete(igm.ng.Name)
		if err != nil {
			return err
		}
		return nil
	} else {
		op, err := igm.conn.adjustNoodPool(igm.ng)
		if err != nil {
			return err
		}
		if err = igm.conn.waitForZoneOperation(op); err != nil {
			return err
		}
	}
	igm.ng.Status.Replicas = *igm.ng.Spec.Replicas
	_, err := store.StoreProvider.MachineSet(igm.conn.Cluster.Name).UpdateStatus(igm.ng)

	return err
}
