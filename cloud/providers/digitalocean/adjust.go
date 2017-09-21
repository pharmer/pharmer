package digitalocean

import (
	"fmt"
	"strconv"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeGroupManager struct {
	cm       *ClusterManager
	instance Instance
	im       *instanceManager
}

func (igm *NodeGroupManager) AdjustNodeGroup(rt api.RunType) (acts []api.Action, err error) {
	acts = make([]api.Action, 0)
	instanceGroupName := igm.cm.namer.GetNodeGroupName(igm.instance.Type.Sku) //igm.cm.ctx.Name + "-" + strings.Replace(igm.instance.Type.Sku, "_", "-", -1) + "-node"

	adjust, _ := Mutator(igm.cm.ctx, igm.cm.cluster, igm.instance, instanceGroupName)
	message := ""
	var action api.ActionType
	if adjust == 0 {
		message = "No changed will be applied"
		action = api.ActionNOP
	} else if adjust < 0 {
		message = fmt.Sprintf("%v node will be deleted from %v group", -adjust, instanceGroupName)
		action = api.ActionDelete
	} else {
		message = fmt.Sprintf("%v node will be added to %v group", adjust, instanceGroupName)
		action = api.ActionAdd
	}
	acts = append(acts, api.Action{
		Action:   action,
		Resource: "Node",
		Message:  message,
	})
	if adjust == 0 || rt == api.DryRun {
		return
	}
	igm.cm.cluster.Generation = igm.instance.Type.ContextVersion
	if adjust == igm.instance.Stats.Count {
		err = igm.createNodeGroup(igm.instance.Stats.Count)
		if err != nil {
			return
		}
	} else if igm.instance.Stats.Count == 0 {
		err = igm.deleteNodeGroup(igm.instance.Type.Sku)
		if err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			return
		}
	} else {
		err = igm.updateNodeGroup(igm.instance.Type.Sku, adjust)
		if err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			return
		}

	}
	return
}

func (igm *NodeGroupManager) createNodeGroup(count int64) error {
	for i := int64(0); i < count; i++ {
		_, err := igm.StartNode()
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()

		}
	}
	return nil
}

func (igm *NodeGroupManager) deleteNodeGroup(sku string) error {
	found, instances, err := igm.im.GetNodeGroup(igm.cm.namer.GetNodeGroupName(sku))
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if !found {
		return errors.New("Instance group not found").Err()
	}
	for _, instance := range instances {
		dropletID, err := strconv.Atoi(instance.Status.ExternalID)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
		err = igm.cm.deleteDroplet(dropletID)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	return nil
}

func (igm *NodeGroupManager) updateNodeGroup(sku string, count int64) error {
	found, instances, err := igm.im.GetNodeGroup(igm.cm.namer.GetNodeGroupName(sku))
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if !found {
		return errors.New("Instance group not found").Err()
	}
	if count < 0 {
		for _, instance := range instances {
			dropletID, _ := strconv.Atoi(instance.Status.ExternalID)
			igm.cm.deleteDroplet(dropletID)
			count++
			if count >= 0 {
				return nil
			}
		}
	} else {
		for i := int64(0); i < count; i++ {
			igm.StartNode()
		}
	}
	return nil
}

func (igm *NodeGroupManager) listInstances(sku string) ([]*api.Node, error) {
	instances := make([]*api.Node, 0)
	kc, err := NewAdminClient(igm.cm.ctx, igm.cm.cluster)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return instances, errors.FromErr(err).WithContext(igm.cm.ctx).Err()

	}
	_, droplets, err := igm.im.GetNodeGroup(igm.cm.namer.GetNodeGroupName(sku))
	if err != nil {
		return instances, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return instances, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	for _, n := range nodes.Items {
		nl := api.FromMap(n.GetLabels())
		if nl.GetString(api.NodeLabelKey_SKU) == sku && nl.GetString(api.NodeLabelKey_Role) == "node" {
			if _, found := droplets[n.Name]; found {
				instances = append(instances, droplets[n.Name])
			}
		}
	}
	return instances, nil
}

func (igm *NodeGroupManager) StartNode() (*api.Node, error) {
	droplet, err := igm.im.createInstance(igm.cm.namer.GenNodeName(igm.instance.Type.Sku), api.RoleNode, igm.instance.Type.Sku)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	// record nodes
	igm.cm.conn.waitForInstance(droplet.ID, "active")
	node, err := igm.im.newKubeInstance(droplet.ID)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	igm.im.applyTag(droplet.ID)
	node.Spec.Role = api.RoleNode
	Store(igm.cm.ctx).Instances(igm.cm.cluster.Name).Create(node)
	return node, nil
}
