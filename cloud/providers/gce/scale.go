package gce

import (
	"fmt"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
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
	inst := cloud.Instance{
		Type: cloud.InstanceType{
			ContextVersion: cm.cluster.Generation,
			Sku:            req.Sku,

			Master:       false,
			SpotInstance: false,
		},
		Stats: cloud.GroupStats{
			Count: req.Count,
		},
	}
	fmt.Println(inst)
	//nodeAdjust, _ := cloud.Mutator(cm.ctx, cm.cluster, inst)
	//igm := &NodeSetManager{
	//	cm:       cm,
	//	instance: inst,
	//}
	//fmt.Println(igm)
	//igm.AdjustNodeSet()
	//flag := false
	//for x := range cm.cluster.Spec.NodeSets {
	//	if cm.cluster.Spec.NodeSets[x].SKU == req.Sku {
	//		cm.cluster.Spec.NodeSets[x].Count += nodeAdjust
	//		flag = true
	//		//fmt.Println(ctx.NodeSets[k].Count, "*********************************>>")
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
	//	cm.cluster.Spec.NodeSets = append(cm.cluster.Spec.NodeSets, ig)
	//}

	//instances, err := igm.cm.listInstances(igm.cm.namer.NodeSetName(req.Sku))
	//if err != nil {
	//	igm.cm.cluster.Status.Reason = err.Error()
	//	//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	//}
	//cloud.AdjustDbInstance(igm.cm.ctx, igm.cm.ins, instances, req.Sku)

	cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	return nil
}

func (cm *ClusterManager) checkNodeSet(instanceGroupName string) bool {
	_, err := cm.conn.computeService.InstanceGroupManagers.Get(cm.cluster.Spec.Cloud.Project, cm.cluster.Spec.Cloud.Zone, instanceGroupName).Do()
	if err != nil {
		return false
	}
	return true
}
