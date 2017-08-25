package gce

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
	fmt.Println(cm.cluster.Spec.ResourceVersion, "*****************")
	cm.namer = namer{cluster: cm.cluster}
	//purchasePHIDs := cm.ctx.Metadata["PurchasePhids"].([]string)
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances, _ = cm.ctx.Store().Instances(cm.cluster.Name).List(api.ListOptions{})

	if req.ApplyToMaster {
		for _, instance := range cm.ins.Instances {
			if instance.Spec.Role == api.RoleKubernetesMaster {
				cm.masterUpdate(instance.Status.ExternalIP, instance.Name, req.KubernetesVersion)
			}
		}
	}
	for _, instance := range cm.ins.Instances {
		if instance.Spec.Role == api.RoleKubernetesPool {
			cm.nodeUpdate(instance.Name)
		}
	}
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
	}
	fmt.Println(igm)
	igm.AdjustInstanceGroup()
	flag := false
	for x := range cm.cluster.Spec.NodeGroups {
		if cm.cluster.Spec.NodeGroups[x].SKU == req.Sku {
			cm.cluster.Spec.NodeGroups[x].Count += nodeAdjust
			flag = true
			//fmt.Println(ctx.NodeGroups[k].Count, "*********************************>>")
		}
		//ctx.NumNodes += v.Count
		//fmt.Println(k.String(), " = ", v.Count)
	}
	if !flag {
		ig := &api.InstanceGroup{
			SKU:              req.Sku,
			Count:            req.Count,
			UseSpotInstances: false,
		}
		cm.cluster.Spec.NodeGroups = append(cm.cluster.Spec.NodeGroups, ig)
	}

	if err := cloud.WaitForReadyNodes(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances, err := igm.cm.listInstances(igm.cm.namer.InstanceGroupName(req.Sku))
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		//return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	cloud.AdjustDbInstance(igm.cm.ctx, igm.cm.ins, instances, req.Sku)

	cm.ctx.Store().Clusters().Update(cm.cluster)
	return nil
}

func (cm *ClusterManager) checkInstanceGroup(instanceGroupName string) bool {
	_, err := cm.conn.computeService.InstanceGroupManagers.Get(cm.cluster.Spec.Project, cm.cluster.Spec.Zone, instanceGroupName).Do()
	if err != nil {
		return false
	}
	return true
}
