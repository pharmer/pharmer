package cloud

import (
	"context"
	"fmt"

	"github.com/appscode/go/crypto/rand"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type GenericNodeGroupManager struct {
	ctx     context.Context
	ng      *api.NodeGroup
	im      InstanceManager
	kc      kubernetes.Interface
	cluster *api.Cluster
	token   string
	// preHook is run once before a set of nodes are added. This can be used to create or update startup scripts. Since this will be
	// called, nodes are added, make sure this method can handle create/update scenarios for a NodeGroup.
	preHook HookFunc
	// gcHook is used to garbage collect when all nodes of a NodeGroup is deleted. This can be used to delete things like startup script.
	gcHook HookFunc
}

var _ NodeGroupManager = &GenericNodeGroupManager{}

func NewNodeGroupManager(ctx context.Context, ng *api.NodeGroup, im InstanceManager, kc kubernetes.Interface, cluster *api.Cluster, token string, initHook HookFunc, gcHook HookFunc) NodeGroupManager {
	return &GenericNodeGroupManager{
		ctx:     ctx,
		ng:      ng,
		im:      im,
		kc:      kc,
		cluster: cluster,
		token:   token,
		preHook: initHook,
		gcHook:  gcHook,
	}
}

func (igm *GenericNodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
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

	igm.ng, err = Store(igm.ctx).NodeGroups(igm.ng.ClusterName).UpdateStatus(igm.ng)
	if err != nil {
		return
	}

	if igm.ng.DeletionTimestamp != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "NodeGroup",
			Message:  fmt.Sprintf("%v node will be deleted from %v group", igm.ng.Spec.Nodes, igm.ng.Name),
		})
		if !dryRun {
			err = igm.DeleteNodes(nodes.Items)
			if err != nil {
				return
			}
			err = Store(igm.ctx).NodeGroups(igm.ng.ClusterName).Delete(igm.ng.Name)
			if err != nil {
				return
			}
		}
	} else if igm.ng.Spec.Nodes == igm.ng.Status.Nodes {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "NodeGroup",
			Message:  fmt.Sprintf("No changed required for node group %s", igm.ng.Name),
		})
		return
	} else if igm.ng.Spec.Nodes < igm.ng.Status.Nodes {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "NodeGroup",
			Message:  fmt.Sprintf("%v node will be deleted from %v group", igm.ng.Status.Nodes-igm.ng.Spec.Nodes, igm.ng.Name),
		})
		if !dryRun {
			err = igm.DeleteNodes(nodes.Items[igm.ng.Spec.Nodes:])
			if err != nil {
				return
			}
			igm.ng.Status.Nodes = igm.ng.Spec.Nodes
			igm.ng, err = Store(igm.ctx).NodeGroups(igm.ng.ClusterName).UpdateStatus(igm.ng)
			if err != nil {
				return
			}
			if igm.ng.Spec.Nodes == 0 && igm.gcHook != nil {
				err = igm.gcHook()
				if err != nil {
					return
				}
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "NodeGroup",
			Message:  fmt.Sprintf("%v node will be added to %v group", igm.ng.Spec.Nodes-igm.ng.Status.Nodes, igm.ng.Name),
		})
		if !dryRun {
			if igm.preHook != nil {
				err = igm.preHook()
				if err != nil {
					return
				}
			}

			err = igm.AddNodes(igm.ng.Spec.Nodes - igm.ng.Status.Nodes)
			if err != nil {
				return
			}

			igm.ng.Status.Nodes = igm.ng.Spec.Nodes
			igm.ng, err = Store(igm.ctx).NodeGroups(igm.ng.ClusterName).UpdateStatus(igm.ng)
			if err != nil {
				return
			}
		}
	}
	return
}

func (igm *GenericNodeGroupManager) AddNodes(count int64) error {
	for i := int64(0); i < count; i++ {
		_, err := igm.im.CreateInstance(rand.WithUniqSuffix(igm.ng.Name), igm.token, igm.ng)
		if err != nil {
			return err
		}
	}
	return nil
}

func (igm *GenericNodeGroupManager) DeleteNodes(nodes []core.Node) error {
	nd, err := NewNodeDrain(igm.ctx, igm.kc, igm.cluster)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		// Drain Node
		nd.Node = node.Name
		if err = nd.Apply(); err != nil {
			return err
		}

		if err = igm.im.DeleteInstanceByProviderID(node.Spec.ProviderID); err != nil {
			return err
		}
		if err = nd.DeleteNode(); err != nil {
			// Sometimes it is not necessary to delete node using api call, because dropping the physical node
			// removes it from node list
			Logger(igm.ctx).Infof("warning: %v", err)
			//return err
		}
	}
	return nil
}
