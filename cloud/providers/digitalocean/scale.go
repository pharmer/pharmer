package digitalocean

import (
	"fmt"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
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

	//purchasePHIDs := cm.ctx.Metadata["PurchasePhids"].([]string)
	cm.namer = namer{cluster: cm.cluster}
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances, _ = cm.ctx.Store().Instances().LoadInstances(cm.cluster.Name)
	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	inst := cloud.Instance{
		Type: cloud.InstanceType{
			ContextVersion: cm.cluster.Spec.ResourceVersion,
			Sku:            req.Sku,

			Master:       false,
			SpotInstance: false,
		},
		Stats: cloud.GroupStats{
			Count: req.Count,
		},
	}

	fmt.Println(cm.cluster.NodeCount(), "<<<----------")
	nodeAdjust, _ := cloud.Mutator(cm.cluster, inst)
	fmt.Println(cm.cluster.NodeCount(), "------->>>>>>>>")
	igm := &InstanceGroupManager{
		cm:       cm,
		instance: inst,
		im:       im,
	}
	igm.AdjustInstanceGroup()

	flag := false
	for x := range cm.cluster.Spec.NodeGroups {
		if cm.cluster.Spec.NodeGroups[x].Sku == req.Sku {
			cm.cluster.Spec.NodeGroups[x].Count += nodeAdjust
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
		cm.cluster.Spec.NodeGroups = append(cm.cluster.Spec.NodeGroups, ig)
	}

	if err := cloud.WaitForReadyNodes(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances, err := igm.listInstances(req.Sku)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	cloud.AdjustDbInstance(cm.ctx, cm.ins, instances, req.Sku)

	cm.ctx.Store().Clusters().SaveCluster(cm.cluster)
	return nil
}
