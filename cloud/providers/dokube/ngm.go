package dokube

import (
	"context"
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type DokubeNodeGroupManager struct {
	ctx   context.Context
	conn  *cloudConnector
	ng    *clusterapi.MachineSet
	owner string
}

func NewDokubeNodeGroupManager(ctx context.Context, conn *cloudConnector, ng *clusterapi.MachineSet, owner string) *DokubeNodeGroupManager {
	return &DokubeNodeGroupManager{ctx: ctx, conn: conn, ng: ng, owner: owner}
}

func (igm *DokubeNodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
	np, err := igm.conn.getNodePool(igm.ng)
	if np == nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Node pool",
			Message:  fmt.Sprintf("Node pool %v will be created", igm.ng.Name),
		})
		if !dryRun {
			if err = igm.conn.addNodePool(igm.ng); err != nil {
				return acts, err
			}
		}

	} else if *igm.ng.Spec.Replicas == 0 || igm.ng.DeletionTimestamp != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Node pool",
			Message:  fmt.Sprintf("Node pool %v will be deleted", igm.ng.Name),
		})
		if !dryRun {
			if err = igm.conn.deleteNodePool(igm.ng); err != nil {
				return acts, err
			}
			err = Store(igm.ctx).Owner(igm.owner).MachineSet(igm.conn.cluster.Name).Delete(igm.ng.Name)
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
			if err = igm.conn.adjustNodePool(igm.ng); err != nil {
				return acts, err
			}
		}
	}
	igm.ng.Status.Replicas = *igm.ng.Spec.Replicas
	_, err = Store(igm.ctx).Owner(igm.owner).MachineSet(igm.conn.cluster.Name).UpdateStatus(igm.ng)
	if err != nil {
		return nil, err
	}
	return acts, err
}
