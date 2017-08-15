package digitalocean

import (
	"fmt"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
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

	//purchasePHIDs := cm.ctx.Metadata["PurchasePhids"].([]string)
	cm.namer = namer{ctx: cm.ctx}
	cm.ins, err = lib.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Load()
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}

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
		im:       im,
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
		ig := &api.InstanceGroup{
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
	instances, err := igm.listInstances(req.Sku)
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	lib.AdjustDbInstance(cm.ins, instances, req.Sku)

	cm.ctx.Save()
	return nil
}
