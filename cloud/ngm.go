package cloud

import (
	"context"
	"fmt"

	"github.com/appscode/go/crypto/rand"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	drain "k8s.io/kubernetes/pkg/kubectl/cmd"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type GenericNodeGroupManager struct {
	ctx     context.Context
	ng      *api.NodeGroup
	im      InstanceManager
	kc      kubernetes.Interface
	cluster *api.Cluster
}

var _ NodeGroupManager = &GenericNodeGroupManager{}

func NewNodeGroupManager(ctx context.Context, ng *api.NodeGroup, im InstanceManager, kc kubernetes.Interface) NodeGroupManager {
	return &GenericNodeGroupManager{ctx: ctx, ng: ng, im: im, kc: kc}
}

func (igm *GenericNodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
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

	if igm.ng.Spec.Nodes == igm.ng.Status.FullyLabeledNodes {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "NodeGroup",
			Message:  fmt.Sprintf("No changed required for node group %s", igm.ng.Name),
		})
		return
	} else if igm.ng.Spec.Nodes < igm.ng.Status.FullyLabeledNodes {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "NodeGroup",
			Message:  fmt.Sprintf("%v node will be deleted from %v group", igm.ng.Status.FullyLabeledNodes-igm.ng.Spec.Nodes, igm.ng.Name),
		})
		if !dryRun {
			err = igm.DeleteNodes(nodes.Items[igm.ng.Spec.Nodes:])
			if err != nil {
				return
			}
			igm.ng, err = Store(igm.ctx).NodeGroups(igm.ng.ClusterName).UpdateStatus(igm.ng)
			if err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "NodeGroup",
			Message:  fmt.Sprintf("%v node will be added to %v group", igm.ng.Spec.Nodes-igm.ng.Status.FullyLabeledNodes, igm.ng.Name),
		})
		if !dryRun {
			err = igm.AddNodes(igm.ng.Spec.Nodes - igm.ng.Status.FullyLabeledNodes)
			if err != nil {
				return
			}

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
		_, err := igm.im.CreateInstance(rand.WithUniqSuffix(igm.ng.Name), igm.ng)
		if err != nil {
			return err
		}
	}
	return nil
}

func (igm *GenericNodeGroupManager) DeleteNodes(nodes []core.Node) error {
	do := drain.DrainOptions{
		Force:              true,
		IgnoreDaemonsets:   true,
		DeleteLocalData:    true,
		GracePeriodSeconds: -1,
		Timeout:            0,
	}
	conf, err := NewClientConfig(igm.ctx, igm.cluster)
	if err != nil {
		return err
	}
	var clientConfig clientcmd.ClientConfig
	clientConfig = clientcmd.NewDefaultClientConfig(conf, &clientcmd.ConfigOverrides{})
	factory := cmdutil.NewFactory(clientConfig)
	do.Factory = factory
	cmdNamespace, _, err := do.Factory.DefaultNamespace()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		// TODO: Drain Node
		do.Factory.NewBuilder(true).
			NamespaceParam(cmdNamespace).DefaultNamespace().
			ResourceNames("node", node.Name).
			Do()
		do.RunDrain()
		err := igm.im.DeleteInstanceByProviderID(node.Spec.ProviderID)
		if err != nil {
			return err
		}
		if igm.kc != nil {
			err := igm.kc.CoreV1().Nodes().Delete(node.Name, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}
