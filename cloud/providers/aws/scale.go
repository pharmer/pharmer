package aws

import (
	"fmt"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	. "github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) Scale(req *proto.ClusterReconfigureRequest) error {
	var err error
	if cm.conn == nil {
		cm.conn, err = NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}

	cm.namer = namer{cluster: cm.cluster}

	inst := Instance{
		Type: InstanceType{
			ContextVersion: cm.cluster.Generation,
			Sku:            req.Sku,

			Master:       false,
			SpotInstance: false,
		},
		Stats: GroupStats{
			Count: req.Count,
		},
	}

	nodeAdjust, _ := Mutator(cm.ctx, cm.cluster, inst, "")
	igm := &NodeGroupManager{
		cm:       cm,
		instance: inst,
	}
	igm.AdjustNodeGroup()
	fmt.Println(nodeAdjust)

	//flag := false
	//for x := range cm.cluster.Spec.NodeGroups {
	//	if cm.cluster.Spec.NodeGroups[x].SKU == req.Sku {
	//		cm.cluster.Spec.NodeGroups[x].Count += nodeAdjust
	//		flag = true
	//		//fmt.Println(ctx.NodeGroups[k].Count, "*********************************>>")
	//	}
	//	//ctx.NumNodes += v.Count
	//	//fmt.Println(k.String(), " = ", v.Count)
	//}
	//if !flag {
	//	ig := &api.IG{
	//		SKU:           req.Sku,
	//		Count:         req.Count,
	//		SpotInstances: false,
	//	}
	//	cm.cluster.Spec.NodeGroups = append(cm.cluster.Spec.NodeGroups, ig)
	//}

	//instances, err := igm.listInstances(cm.namer.AutoScalingGroupName(req.Sku))
	//if err != nil {
	//	igm.cm.cluster.Status.Reason = err.Error()
	//	//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	//}
	//AdjustDbInstance(igm.cm.ctx, igm.cm.ins, instances, req.Sku)

	Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	return nil
}
