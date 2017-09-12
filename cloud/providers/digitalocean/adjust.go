package digitalocean

import (
	gtx "context"
	"fmt"
	"strconv"
	"strings"

	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/digitalocean/godo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeGroupManager struct {
	cm       *ClusterManager
	instance Instance
	im       *instanceManager
}

func (igm *NodeGroupManager) AdjustNodeGroup() error {
	instanceGroupName := igm.cm.namer.GetNodeGroupName(igm.instance.Type.Sku) //igm.cm.ctx.Name + "-" + strings.Replace(igm.instance.Type.Sku, "_", "-", -1) + "-node"
	found, _, err := igm.GetNodeGroup(instanceGroupName)
	if err != nil {
		igm.cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	fmt.Println(found)

	igm.cm.cluster.Generation = igm.instance.Type.ContextVersion
	igm.cm.cluster, _ = Store(igm.cm.ctx).Clusters().Get(igm.cm.cluster.Name)
	var nodeAdjust int64 = 0
	if found {
		nodeAdjust, _ = Mutator(igm.cm.ctx, igm.cm.cluster, igm.instance, instanceGroupName)
	}
	if !found {
		err = igm.createNodeGroup(igm.instance.Stats.Count)
	} else if igm.instance.Stats.Count == 0 {
		if nodeAdjust < 0 {
			nodeAdjust = -nodeAdjust
		}
		err := igm.deleteNodeGroup(igm.instance.Type.Sku, nodeAdjust)
		if err != nil {
			igm.cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	} else {
		if nodeAdjust < 0 {
			err := igm.deleteNodeGroup(igm.instance.Type.Sku, -nodeAdjust)
			if err != nil {
				igm.cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
		} else {
			err := igm.createNodeGroup(nodeAdjust)
			if err != nil {
				igm.cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
		}
	}
	return nil
}

func (igm *NodeGroupManager) GetNodeGroup(instanceGroup string) (bool, map[string]*api.Node, error) {
	var flag bool = false
	igm.im.conn.client.Droplets.List(gtx.TODO(), &godo.ListOptions{})
	existingNGs := make(map[string]*api.Node)
	droplets, _, err := igm.cm.conn.client.Droplets.List(gtx.TODO(), &godo.ListOptions{})
	if err != nil {
		return flag, existingNGs, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	for _, item := range droplets {
		if strings.HasPrefix(item.Name, instanceGroup) {
			flag = true
			instance, err := igm.im.newKubeInstance(item.ID)
			if err != nil {
				return flag, existingNGs, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
			}
			instance.Spec.Role = api.RoleNode
			internalIP, err := item.PrivateIPv4()
			existingNGs[internalIP] = instance
		}

	}
	return flag, existingNGs, nil
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

func (igm *NodeGroupManager) deleteNodeGroup(sku string, count int64) error {
	instances, err := igm.listInstances(sku)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	for _, instance := range instances {
		count--
		dropletID, err := strconv.Atoi(instance.Status.ExternalID)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
		err = igm.cm.deleteDroplet(dropletID, instance.Status.PrivateIP)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
		if count <= 0 {
			break
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
	_, droplets, err := igm.GetNodeGroup(igm.cm.namer.GetNodeGroupName(sku))
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
