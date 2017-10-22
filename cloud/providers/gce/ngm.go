package gce

import (
	"context"
	"fmt"

	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type GCENodeGroupManager struct {
	ctx   context.Context
	conn  *cloudConnector
	namer namer
	ng    *api.NodeGroup
	kc    kubernetes.Interface
}

func NewGCENodeGroupManager(ctx context.Context, conn *cloudConnector, namer namer, ng *api.NodeGroup, kc kubernetes.Interface) *GCENodeGroupManager {
	return &GCENodeGroupManager{ctx: ctx, conn: conn, namer: namer, ng: ng, kc: kc}
}

func (igm *GCENodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
	nodes := &core.NodeList{}
	if igm.kc != nil {
		nodes, err = igm.kc.CoreV1().Nodes().List(metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				api.NodeLabelKey_NodeGroup: igm.ng.Name,
			}).String(),
		})
		if err != nil {
			return
		}
	}

	igm.ng.Status.FullyLabeledNodes = int64(len(nodes.Items))
	igm.ng.Status.ObservedGeneration = igm.ng.Generation
	igm.ng.Spec.Template.Spec.DiskType = "pd-standard"
	adjust := igm.ng.Spec.Nodes - igm.ng.Status.FullyLabeledNodes

	if (igm.ng.DeletionTimestamp != nil || igm.ng.Spec.Nodes == 0) && igm.ng.Status.FullyLabeledNodes > 0 {
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
			if err = igm.conn.deleteOnlyNodeGroup(igm.ng.Name, instanceTemplate); err != nil {
				return
			}
			Store(igm.ctx).NodeGroups(igm.ng.ClusterName).Delete(igm.ng.Name)
		}

	} else if igm.ng.Spec.Nodes == igm.ng.Status.FullyLabeledNodes {
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
			if op2, err = igm.conn.createNodeInstanceTemplate(igm.ng); err != nil {
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

		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Autoscaler",
			Message:  fmt.Sprintf("Autoscaler %v will be created", igm.ng.Name),
		})
		if !dryRun {
			var op4 string
			if op4, err = igm.conn.createAutoscaler(igm.ng); err != nil {
				return
			} else {
				if err = igm.conn.waitForZoneOperation(op4); err != nil {
					return
				}
			}
		}
	} else {
		if err = igm.conn.updateNodeGroup(igm.ng, igm.ng.Spec.Nodes); err != nil {
			return
		}
	}

	return
}
