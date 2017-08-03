package aws

import (
	"fmt"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/cloud/common"
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

	cm.namer = namer{ctx: cm.ctx}
	cm.ins, err = common.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Load()

	inst := common.Instance{
		Type: common.InstanceType{
			ContextVersion: cm.ctx.ContextVersion,
			Sku:            req.Sku,

			Master:       false,
			SpotInstance: false,
		},
		Stats: common.GroupStats{
			Count: req.Count,
		},
	}

	fmt.Println(cm.ctx.NodeCount(), "<<<----------")
	nodeAdjust, _ := common.Mutator(cm.ctx, inst)
	fmt.Println(cm.ctx.NodeCount(), "------->>>>>>>>")
	igm := &InstanceGroupManager{
		cm:       cm,
		instance: inst,
	}
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

	if err := common.WaitForReadyNodes(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances, err := igm.listInstances(cm.namer.AutoScalingGroupName(req.Sku))
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	common.AdjustDbInstance(igm.cm.ins, instances, req.Sku)

	cm.ctx.Save()
	return nil
}
