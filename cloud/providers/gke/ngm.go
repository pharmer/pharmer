package gke

import (
	"context"
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	container "google.golang.org/api/container/v1"
)

type GKENodeGroupManager struct {
	ctx  context.Context
	conn *cloudConnector
	ng   *api.NodeGroup
}

func NewGKENodeGroupManager(ctx context.Context, conn *cloudConnector, ng *api.NodeGroup) *GKENodeGroupManager {
	return &GKENodeGroupManager{ctx: ctx, conn: conn, ng: ng}
}

func (igm *GKENodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
	var np *container.NodePool
	var op string
	np, _ = igm.conn.containerService.Projects.Zones.Clusters.NodePools.Get(igm.conn.cluster.Spec.Cloud.Project, igm.conn.cluster.Spec.Cloud.Zone, igm.conn.cluster.Name, igm.ng.Name).Do()

	if np == nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Node pool",
			Message:  fmt.Sprintf("Node pool %v will be created", igm.ng.Name),
		})
		if !dryRun {
			if op, err = igm.conn.addNodePool(igm.ng); err != nil {
				return acts, err
			}
			if err = igm.conn.waitForZoneOperation(op); err != nil {
				return acts, err
			}
		}

	} else if igm.ng.Spec.Nodes == 0 || igm.ng.DeletionTimestamp != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Node pool",
			Message:  fmt.Sprintf("Node pool %v will be deleted", igm.ng.Name),
		})
		if !dryRun {
			if op, err = igm.conn.deleteNoodPool(igm.ng); err != nil {
				return acts, err
			}
			if err = igm.conn.waitForZoneOperation(op); err != nil {
				return acts, err
			}
			err = Store(igm.ctx).NodeGroups(igm.conn.cluster.Name).Delete(igm.ng.Name)
			if err != nil {
				return acts, err
			}
			return
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionUpdate,
			Resource: "Node pool",
			Message:  fmt.Sprintf("Node pool %v will be updated", igm.ng.Name),
		})
		if !dryRun {
			if op, err = igm.conn.adjustNoodPool(igm.ng); err != nil {
				return acts, err
			}
			if err = igm.conn.waitForZoneOperation(op); err != nil {
				return acts, err
			}
		}
	}
	igm.ng.Status.Nodes = igm.ng.Spec.Nodes
	Store(igm.ctx).NodeGroups(igm.conn.cluster.Name).UpdateStatus(igm.ng)

	return acts, err
}
