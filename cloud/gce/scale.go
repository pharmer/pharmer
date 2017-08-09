package gce

import (
	"fmt"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/contexts"
)

func (cm *clusterManager) scale(req *proto.ClusterReconfigureRequest) error {
	var err error
	if cm.conn == nil {
		cm.conn, err = NewConnector(cm.ctx)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	fmt.Println(cm.ctx.ContextVersion, "*****************")
	cm.namer = namer{ctx: cm.ctx}
	//purchasePHIDs := cm.ctx.Metadata["PurchasePhids"].([]string)
	cm.ins, err = lib.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Load()

	inst := lib.Instance{
		Type: lib.InstanceType{
			ContextVersion: cm.ctx.ContextVersion,
			Sku:            req.Sku,

			Master:       false,
			SpotInstance: false,
		},
		Stats: lib.GroupStats{
			Count: req.Count,
		},
	}
	fmt.Println(cm.ctx.NodeCount(), "<<<----------")
	nodeAdjust, _ := lib.Mutator(cm.ctx, inst)
	fmt.Println(cm.ctx.NodeCount(), "------->>>>>>>>")
	igm := &InstanceGroupManager{
		cm:       cm,
		instance: inst,
	}
	fmt.Println(igm)
	igm.AdjustInstanceGroup()
	flag := false
	for x := range cm.ctx.NodeGroups {
		if cm.ctx.NodeGroups[x].Sku == req.Sku {
			cm.ctx.NodeGroups[x].Count += nodeAdjust
			flag = true
			//fmt.Println(ctx.NodeGroups[k].Count, "*********************************>>")
		}
		//ctx.NumNodes += v.Count
		//fmt.Println(k.String(), " = ", v.Count)
	}
	if !flag {
		ig := &contexts.InstanceGroup{
			Sku:              req.Sku,
			Count:            req.Count,
			UseSpotInstances: false,
		}
		cm.ctx.NodeGroups = append(cm.ctx.NodeGroups, ig)
	}

	if err := lib.WaitForReadyNodes(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances, err := igm.cm.listInstances(igm.cm.namer.InstanceGroupName(req.Sku))
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	lib.AdjustDbInstance(igm.cm.ins, instances, req.Sku)

	cm.ctx.Save()
	return nil
}

func (cm *clusterManager) checkInstanceGroup(instanceGroupName string) bool {
	_, err := cm.conn.computeService.InstanceGroupManagers.Get(cm.ctx.Project, cm.ctx.Zone, instanceGroupName).Do()
	if err != nil {
		return false
	}
	return true
}
