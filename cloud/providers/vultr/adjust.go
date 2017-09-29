package vultr

import (
	"fmt"
	"strconv"
	"time"

	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
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
	if adjust == 0 {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Node",
			Message:  "No changed will be applied",
		})
		return
	}
	igm.cm.cluster.Generation = igm.instance.Type.ContextVersion
	if adjust == igm.instance.Stats.Count {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Node",
			Message:  fmt.Sprintf("%v node will be added to %v group", igm.instance.Stats.Count, instanceGroupName),
		})
		if rt != api.DryRun {
			if err = igm.upsertNodeGroup(igm.instance.Stats.Count); err != nil {
				return
			}
		}
	} else if igm.instance.Stats.Count == 0 {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Node",
			Message:  fmt.Sprintf("Node group %v will be deleted", instanceGroupName),
		})
		if rt != api.DryRun {
			if err = igm.deleteNodeGroup(igm.instance.Type.Sku); err != nil {
				igm.cm.cluster.Status.Reason = err.Error()
				return
			}
		}
	} else {
		var message string
		var action api.ActionType
		if adjust < 0 {
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
		if rt != api.DryRun {
			if err = igm.updateNodeGroup(igm.instance.Type.Sku, adjust); err != nil {
				igm.cm.cluster.Status.Reason = err.Error()
				return
			}
		}

	}
	return
}

// start nodes
type NodeInfo struct {
	nodeId string
	state  int
}

func (igm *NodeGroupManager) upsertNodeGroup(count int64) (err error) {
	var nodeScriptID int
	var found bool
	if found, nodeScriptID, err = igm.im.getStartupScript(igm.instance.Type.Sku, api.RoleNode); err != nil {
		return
	}
	if !found {
		if nodeScriptID, err = igm.im.createStartupScript(igm.instance.Type.Sku, api.RoleNode); err != nil {
			return errors.FromErr(err).WithContext(igm.im.ctx).Err()
		}
	}

	nodes := make([]*NodeInfo, 0)
	for i := int64(0); i < count; i++ {
		nodeID, err := igm.im.createInstance(igm.im.namer.GenNodeName(igm.instance.Type.Sku), igm.instance.Type.Sku, nodeScriptID)
		if err != nil {
			igm.im.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(igm.im.ctx).Err()
		}
		nodes = append(nodes, &NodeInfo{
			nodeId: nodeID,
			state:  0,
		})
	}
	/*if err = igm.rebootNodes(nodes); err != nil {
		return err
	}*/
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
		err = igm.im.deleteServer(instance.Status.ExternalID)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	_, scriptID, err := igm.im.getStartupScript(sku, api.RoleNode)
	if err != nil {
		return err
	}
	return igm.im.conn.client.DeleteStartupScript(strconv.Itoa(scriptID))
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
			igm.im.deleteServer(instance.Status.ExternalID)
			count++
			if count >= 0 {
				return nil
			}
		}
	} else {
		igm.upsertNodeGroup(count)
	}
	return nil
}

func (igm *NodeGroupManager) rebootNodes(nodes []*NodeInfo) error {
	/*
		Now, for each node,
		- start at state = 0
		- Wait for the node to become PoweredOff and set state = 1
		- For 60 seconds, after it becomes PoweredOff (2 iteration with 30 sec delay)
		- At state = 3, Boot the node
		- Done, when all nodes had a chance to Boot
	*/
	var done bool
	for true {
		time.Sleep(30 * time.Second)
		done = true
		for _, info := range nodes {
			if info.state == 0 {
				server, err := igm.im.conn.client.GetServer(info.nodeId)
				// ignore error, and try again
				if err == nil {
					Logger(igm.im.ctx).Infof("Instance %v (%v) is %v", server.Name, server.ID, server.Status)
					if server.Status == "active" && server.PowerStatus == "running" {
						Logger(igm.im.ctx).Infof("Instance %v is %v", server.Name, server.Status)
						info.state = 1
						// create node
						node, err := igm.im.newKubeInstance(&server)
						if err != nil {
							igm.im.cluster.Status.Reason = err.Error()
							errors.FromErr(err).WithContext(igm.im.ctx).Err()
							//return
						}
						node.Spec.Role = api.RoleNode
						Store(igm.im.ctx).Instances(igm.im.cluster.Name).Create(node)
					}
				}
			} else {
				info.state++
			}
			if info.state == 3 {
				err := igm.im.conn.client.RebootServer(info.nodeId)
				if err != nil {
					info.state-- // retry on error
				}
			}
			if info.state < 3 {
				done = false
			}
		}
		if done {
			break
		}
	}
	return nil
	//Logger(cm.ctx).Info("Waiting for cluster initialization")
}
