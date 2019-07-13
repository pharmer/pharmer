package gke

import (
	"google.golang.org/api/container/v1"
	"pharmer.dev/pharmer/cloud"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type GKENodeGroupManager struct {
	*cloud.Scope
	conn *cloudConnector
	ng   *clusterapi.MachineSet
}

func NewGKENodeGroupManager(scope *cloud.Scope, conn *cloudConnector, ng *clusterapi.MachineSet) *GKENodeGroupManager {
	return &GKENodeGroupManager{Scope: scope, conn: conn, ng: ng}
}

func (igm *GKENodeGroupManager) Apply() error {
	log := igm.Logger

	var np *container.NodePool
	np, _ = igm.conn.containerService.Projects.Zones.Clusters.NodePools.Get(igm.conn.Cluster.Spec.Config.Cloud.Project, igm.conn.Cluster.Spec.Config.Cloud.Zone, igm.conn.Cluster.Name, igm.ng.Name).Do()

	if np == nil {
		op, err := igm.conn.addNodePool(igm.ng)
		if err != nil {
			log.Error(err, "failed to add node pool")
			return err
		}
		if err := igm.conn.waitForZoneOperation(op); err != nil {
			log.Error(err, "zonal operation failed", "operation", op)
			return err
		}

	} else if *igm.ng.Spec.Replicas == 0 || igm.ng.DeletionTimestamp != nil {
		op, err := igm.conn.deleteNoodPool(igm.ng)
		if err != nil {
			log.Error(err, "failed to delete nodes")
			return err
		}
		if err = igm.conn.waitForZoneOperation(op); err != nil {
			log.Error(err, "zonal operation failed", "operation", op)
			return err
		}
		err = igm.StoreProvider.MachineSet(igm.conn.Cluster.Name).Delete(igm.ng.Name)
		if err != nil {
			log.Error(err, "failed to delete machine from store")
			return err
		}
		return nil
	} else {
		op, err := igm.conn.adjustNodePool(igm.ng)
		if err != nil {
			log.Error(err, "failed to adjust node pool")
			return err
		}
		if err = igm.conn.waitForZoneOperation(op); err != nil {
			log.Error(err, "zonal operation failed", "operation", op)
			return err
		}
	}
	igm.ng.Status.Replicas = *igm.ng.Spec.Replicas
	_, err := igm.StoreProvider.MachineSet(igm.conn.Cluster.Name).UpdateStatus(igm.ng)

	if err != nil {
		log.Error(err, "failed to update machineset in store")
		return err
	}

	return nil
}
