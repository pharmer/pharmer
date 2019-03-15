package gce

import (
	"context"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type GCENodeGroupManager struct {
	ctx   context.Context
	conn  *cloudConnector
	namer namer
	ng    *api.NodeGroup
	kc    kubernetes.Interface
	token string
	owner string
}

func NewGCENodeGroupManager(ctx context.Context, conn *cloudConnector, namer namer, ng *api.NodeGroup, kc kubernetes.Interface, token string) *GCENodeGroupManager {
	return &GCENodeGroupManager{ctx: ctx, conn: conn, namer: namer, ng: ng, kc: kc, token: token}
}

func (igm *GCENodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
	/*
			nodes := &core.NodeList{}
			if igm.kc != nil {
				nodes, err = igm.kc.CoreV1().Nodes().List(metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{
						api.NodePoolKey: igm.ng.Name,
					}).String(),
				})
				if err != nil {
					return
				}
			}

			igm.ng.Status.Nodes = int64(len(nodes.Items))
			igm.ng.Status.ObservedGeneration = igm.ng.Generation
			// igm.ng.Spec.Template.Spec.DiskType = "pd-standard"
			adjust := igm.ng.Spec.Nodes - igm.ng.Status.Nodes

		if (igm.ng.DeletionTimestamp != nil || igm.ng.Spec.Nodes == 0) && igm.ng.Status.Nodes > 0 {
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "Node Group",
				Message:  fmt.Sprintf("No changed required for node group %s", igm.ng.Name),
			})
			instanceTemplate := igm.namer.InstanceTemplateName(igm.ng.Spec.Template.Spec.SKU)
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Instance Template",
				Message:  fmt.Sprintf("Node group %v  with instance template %v will be delete", igm.ng.Name, instanceTemplate),
			})
			if !dryRun {
				var nd NodeDrain
				if nd, err = NewNodeDrain(igm.ctx, igm.kc, igm.conn.cluster, igm.owner); err != nil {
					return
				}
				for _, node := range nodes.Items {
					nd.Node = node.Name
					if err = nd.Apply(); err != nil {
						return
					}
					for _, node := range nodes.Items {
						nd.Node = node.Name
						if err = nd.Apply(); err != nil {
							return
						}
					}
					if err = igm.conn.deleteOnlyNodeGroup(igm.ng.Name, instanceTemplate); err != nil {
						return
					}
					for _, node := range nodes.Items {
						nd.Node = node.Name
						if er := nd.DeleteNode(); er != nil {
							// ignore error while deleting node
							Logger(igm.ctx).Infoln("Failed to delete node ", node.Name, er)
						}
					}
					Store(igm.ctx).NodeGroups(igm.ng.ClusterName).Delete(igm.ng.Name)
				}

			} else if igm.ng.Spec.Nodes == igm.ng.Status.Nodes {
				acts = append(acts, api.Action{
					Action:   api.ActionNOP,
					Resource: "NodeGroup",
					Message:  fmt.Sprintf("No changed required for node group %s", igm.ng.Name),
				})
				return
			} else if adjust == igm.ng.Spec.Nodes {
				acts = append(acts, api.Action{
					Action:   api.ActionAdd,
					Resource: "Instance Template",
					Message:  fmt.Sprintf("Instance template %v will be created", igm.namer.InstanceTemplateName(igm.ng.Spec.Template.Spec.SKU)),
				})
				if !dryRun {
					var op2 string
					if op2, err = igm.conn.createNodeInstanceTemplate(igm.ng, igm.token); err != nil {
						return
					} else {
						if err = igm.conn.waitForGlobalOperation(op2); err != nil {
							return
						}
					}
				}
				acts = append(acts, api.Action{
					Action:   api.ActionAdd,
					Resource: "Node group",
					Message:  fmt.Sprintf("Node group %v will be created", igm.ng.Name),
				})
				if !dryRun {
					var op3 string
					if op3, err = igm.conn.createNodeGroup(igm.ng); err != nil {
						return
					} else {
						if err = igm.conn.waitForZoneOperation(op3); err != nil {
							return
						}
					}
				}
			} else {
				if adjust > 0 {
					if err = igm.conn.addNodeIntoGroup(igm.ng, igm.ng.Spec.Nodes); err != nil {
						return
					}
				} else if adjust < 0 {
					if err = igm.deleteNodeWithDrain(nodes.Items[igm.ng.Spec.Nodes:]); err != nil {
						return
					}
				}
			}
	*/

	return
}

func (igm *GCENodeGroupManager) deleteNodeWithDrain(nodes []core.Node) error {
	/*nd, err := NewNodeDrain((igm.ctx, igm.kc, igm.conn.cluster, igm.owner)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		nd.Node = node.Name
		if err = nd.Apply(); err != nil {
			return err
		}
		if err = igm.conn.deleteGroupInstances(igm.ng, node.Name); err != nil {
			return err
		}
		for _, node := range nodes {
			nd.Node = node.Name
			if err = nd.Apply(); err != nil {
				return err
			}
			if err = igm.conn.deleteGroupInstances(igm.ng, node.Name); err != nil {
				return err
			}
			if err = nd.DeleteNode(); err != nil {
				Logger(igm.ctx).Infoln("Failed to delete node ", node.Name, err)
			}

		}
	}*/
	return nil
}
