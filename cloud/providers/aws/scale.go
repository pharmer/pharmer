package aws

import (
	"fmt"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *clusterManager) scale(req *proto.ClusterReconfigureRequest) error {
	var err error
	if cm.conn == nil {
		cm.conn, err = NewConnector(cm.cluster)
		if err != nil {
			cm.cluster.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}

	cm.namer = namer{cluster: cm.cluster}
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances, _ = cm.ctx.Store().Instances().LoadInstances(cm.cluster.Name)

	inst := cloud.Instance{
		Type: cloud.InstanceType{
			ContextVersion: cm.cluster.ContextVersion,
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
	}
	igm.AdjustInstanceGroup()

	flag := false
	for x := range cm.cluster.NodeGroups {
		if cm.cluster.NodeGroups[x].Sku == req.Sku {
			cm.cluster.NodeGroups[x].Count += nodeAdjust
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
		cm.cluster.NodeGroups = append(cm.cluster.NodeGroups, ig)
	}

	if err := cloud.WaitForReadyNodes(cm.ctx, cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances, err := igm.listInstances(cm.namer.AutoScalingGroupName(req.Sku))
	if err != nil {
		igm.cm.cluster.StatusCause = err.Error()
		//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	cloud.AdjustDbInstance(igm.cm.ctx, igm.cm.ins, instances, req.Sku)

	cm.ctx.Store().Clusters().SaveCluster(cm.cluster)
	return nil
}
