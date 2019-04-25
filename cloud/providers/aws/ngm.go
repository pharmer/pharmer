package aws

// import (
// 	"context"
// 	"fmt"

// 	. "github.com/appscode/go/context"
// 	. "github.com/appscode/go/types"
// 	"github.com/aws/aws-sdk-go/service/autoscaling"
// 	api "github.com/pharmer/pharmer/apis/v1beta1"
// 	. "github.com/pharmer/pharmer/cloud"
// 	"github.com/pkg/errors"
// 	core "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/labels"
// 	"k8s.io/client-go/kubernetes"
// )

// type AWSNodeGroupManager struct {
// 	ctx   context.Context
// 	conn  *cloudConnector
// 	namer namer
// 	ng    *api.NodeGroup
// 	kc    kubernetes.Interface
// 	token string
// }

// func NewAWSNodeGroupManager(ctx context.Context, conn *cloudConnector, namer namer, ng *api.NodeGroup, kc kubernetes.Interface, token string) *AWSNodeGroupManager {
// 	return &AWSNodeGroupManager{ctx: ctx, conn: conn, namer: namer, ng: ng, kc: kc, token: token}
// }

// func (igm *AWSNodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
// 	nodes := &core.NodeList{}
// 	if igm.kc != nil {
// 		nodes, err = igm.kc.CoreV1().Nodes().List(metav1.ListOptions{
// 			LabelSelector: labels.SelectorFromSet(map[string]string{
// 				api.NodePoolKey: igm.ng.Name,
// 			}).String(),
// 		})
// 		if err != nil {
// 			return
// 		}
// 	}
// 	igm.ng.Status.Nodes = int64(len(nodes.Items))
// 	igm.ng.Status.ObservedGeneration = igm.ng.Generation
// 	//igm.ng.Spec.Template.Spec.DiskType = "gp2"

// 	adjust := igm.ng.Spec.Nodes - igm.ng.Status.Nodes

// 	if (igm.ng.DeletionTimestamp != nil || igm.ng.Spec.Nodes == 0) && igm.ng.Status.Nodes > 0 {
// 		acts = append(acts, api.Action{
// 			Action:   api.ActionDelete,
// 			Resource: "Autoscaler",
// 			Message:  fmt.Sprintf("Autoscaler %v  will be delete", igm.ng.Name),
// 		})
// 		if !dryRun {
// 			var nd NodeDrain
// 			if nd, err = NewNodeDrain(igm.ctx, igm.kc, igm.conn.cluster); err != nil {
// 				return
// 			}
// 			for _, node := range nodes.Items {
// 				nd.Node = node.Name
// 				if err = nd.Apply(); err != nil {
// 					return
// 				}
// 			}
// 			if err = igm.conn.deleteAutoScalingGroup(igm.ng.Name); err != nil {
// 				return
// 			}
// 			for _, node := range nodes.Items {
// 				nd.Node = node.Name
// 				if err = nd.DeleteNode(); err != nil {
// 					return
// 				}
// 			}
// 		}
// 		launchConfig := igm.namer.LaunchConfigName(igm.ng.Spec.Template.Spec.SKU)
// 		acts = append(acts, api.Action{
// 			Action:   api.ActionDelete,
// 			Resource: "Launch configuration",
// 			Message:  fmt.Sprintf("Launch configuration %v  will be delete", launchConfig),
// 		})
// 		if !dryRun {
// 			if err = igm.conn.deleteLaunchConfiguration(launchConfig); err != nil {
// 				return
// 			}
// 		}
// 		if !dryRun {
// 			Store(igm.ctx).NodeGroups(igm.ng.ClusterName).Delete(igm.ng.Name)
// 		}
// 	} else if igm.ng.Spec.Nodes == igm.ng.Status.Nodes {
// 		acts = append(acts, api.Action{
// 			Action:   api.ActionNOP,
// 			Resource: "NodeGroup",
// 			Message:  fmt.Sprintf("No changed required for node group %s", igm.ng.Name),
// 		})
// 		return
// 	} else if adjust == igm.ng.Spec.Nodes {
// 		acts = append(acts, api.Action{
// 			Action:   api.ActionAdd,
// 			Resource: "Lunch Configuration",
// 			Message:  fmt.Sprintf("Lunch configuration %v will be created", igm.namer.LaunchConfigName(igm.ng.Spec.Template.Spec.SKU)),
// 		})

// 		acts = append(acts, api.Action{
// 			Action:   api.ActionAdd,
// 			Resource: "Auto scaler",
// 			Message:  fmt.Sprintf("Autoscaler %v will be created", igm.ng.Name),
// 		})
// 		if !dryRun {
// 			if err = igm.createNodeGroup(igm.ng); err != nil {
// 				return
// 			}
// 			Store(igm.ctx).NodeGroups(igm.ng.ClusterName).Update(igm.ng)
// 		}
// 	} else {
// 		if adjust < 0 {
// 			acts = append(acts, api.Action{
// 				Action:   api.ActionDelete,
// 				Resource: "NodeGroup",
// 				Message:  fmt.Sprintf("%v node will be deleted from %v group", igm.ng.Status.Nodes-igm.ng.Spec.Nodes, igm.ng.Name),
// 			})
// 			if !dryRun {
// 				if err = igm.deleteNodeWithDrain(nodes.Items[igm.ng.Spec.Nodes:]); err != nil {
// 					return
// 				}
// 			}
// 		} else if adjust > 0 {
// 			acts = append(acts, api.Action{
// 				Action:   api.ActionAdd,
// 				Resource: "NodeGroup",
// 				Message:  fmt.Sprintf("%v node will be added to %v group", igm.ng.Spec.Nodes-igm.ng.Status.Nodes, igm.ng.Name),
// 			})
// 			if !dryRun {
// 				if err = igm.updateNodeGroup(igm.ng, igm.ng.Spec.Nodes); err != nil {
// 					return
// 				}
// 			}
// 		}
// 		Store(igm.ctx).NodeGroups(igm.ng.ClusterName).Update(igm.ng)

// 	}
// 	// Store(igm.ctx).Clusters().UpdateStatus(igm.cm.cluster)
// 	return
// }

// func (igm *AWSNodeGroupManager) createNodeGroup(ng *api.NodeGroup) error {
// 	launchConfig := igm.namer.LaunchConfigName(ng.Spec.Template.Spec.SKU)

// 	if err := igm.conn.createLaunchConfiguration(launchConfig, igm.token, ng); err != nil {
// 		return errors.Wrap(err, ID(igm.ctx))
// 	}
// 	if err := igm.conn.createAutoScalingGroup(ng.Name, launchConfig, ng.Spec.Nodes); err != nil {
// 		return errors.Wrap(err, ID(igm.ctx))
// 	}
// 	return nil
// }

// func (igm *AWSNodeGroupManager) deleteNodeWithDrain(nodes []core.Node) error {
// 	nd, err := NewNodeDrain(igm.ctx, igm.kc, igm.conn.cluster)
// 	if err != nil {
// 		return err
// 	}

// 	for _, node := range nodes {
// 		nd.Node = node.Name
// 		if err = nd.Apply(); err != nil {
// 			return err
// 		}
// 		instanceID, err := splitProviderID(node.Spec.ProviderID)
// 		if err != nil {
// 			return err
// 		}
// 		if err = igm.conn.deleteGroupInstances(igm.ng, instanceID); err != nil {
// 			return err
// 		}
// 		if err = nd.DeleteNode(); err != nil {
// 			return err
// 		}

// 	}
// 	return nil
// }

// func (igm *AWSNodeGroupManager) updateNodeGroup(ng *api.NodeGroup, size int64) error {
// 	group, err := igm.conn.describeGroupInfo(ng.Name)
// 	if err != nil {
// 		return errors.Wrap(err, ID(igm.ctx))
// 	}
// 	if size > *group.AutoScalingGroups[0].MaxSize {
// 		_, err := igm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
// 			AutoScalingGroupName: StringP(ng.Name),
// 			DefaultCooldown:      Int64P(1),
// 			MaxSize:              Int64P(size),
// 			DesiredCapacity:      Int64P(size),
// 		})
// 		if err != nil {
// 			return errors.Wrap(err, ID(igm.ctx))
// 		}

// 	} else if size < *group.AutoScalingGroups[0].MinSize {
// 		_, err := igm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
// 			AutoScalingGroupName: StringP(ng.Name),
// 			MinSize:              Int64P(size),
// 			DesiredCapacity:      Int64P(size),
// 		})
// 		if err != nil {
// 			return errors.Wrap(err, ID(igm.ctx))
// 		}
// 	} else {
// 		_, err := igm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
// 			AutoScalingGroupName: StringP(ng.Name),
// 			DesiredCapacity:      Int64P(size),
// 		})
// 		if err != nil {
// 			return errors.Wrap(err, ID(igm.ctx))
// 		}
// 	}

// 	//time.Sleep(2 * time.Minute)
// 	return nil
// }
