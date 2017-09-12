package gce

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
	fmt.Println(cm.cluster.Generation, "*****************")
	cm.namer = namer{cluster: cm.cluster}
	// TODO: FixIT!
	//if req.ApplyToMaster {
	//	for _, instance := range cm.ins.Instances {
	//		if instance.Spec.Role == api.RoleKubernetesMaster {
	//			cm.masterUpdate(instance.Status.PublicIP, instance.Name, req.KubernetesVersion)
	//		}
	//	}
	//}
	//for _, instance := range cm.ins.Instances {
	//	if instance.Spec.Role == api.RoleKubernetesPool {
	//		cm.nodeUpdate(instance.Name)
	//	}
	//}
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
	fmt.Println(inst)
	//nodeAdjust, _ := Mutator(cm.ctx, cm.cluster, inst)
	//igm := &NodeGroupManager{
	//	cm:       cm,
	//	instance: inst,
	//}
	//fmt.Println(igm)
	//igm.AdjustNodeGroup()
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

	//instances, err := igm.cm.listInstances(igm.cm.namer.NodeGroupName(req.Sku))
	//if err != nil {
	//	igm.cm.cluster.Status.Reason = err.Error()
	//	//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	//}
	//AdjustDbInstance(igm.cm.ctx, igm.cm.ins, instances, req.Sku)

	Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	return nil
}

func (cm *ClusterManager) checkNodeGroup(instanceGroupName string) bool {
	_, err := cm.conn.computeService.InstanceGroupManagers.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, instanceGroupName).Do()
	if err != nil {
		return false
	}
	return true
}
