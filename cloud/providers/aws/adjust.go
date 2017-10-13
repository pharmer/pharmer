package aws

import (
	"fmt"

	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type AWSNodeGroupManager struct {
	cm *ClusterManager
	ng *api.NodeGroup
	kc kubernetes.Interface
}

func NewAWSNodeGroupManager(cm *ClusterManager, ng *api.NodeGroup, kc kubernetes.Interface) *AWSNodeGroupManager {
	return &AWSNodeGroupManager{cm: cm, ng: ng, kc: kc}
}

func (igm *AWSNodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
	nodes := &apiv1.NodeList{}
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
	igm.ng.Spec.Template.Spec.DiskType = "gp2"

	adjust := igm.ng.Spec.Nodes - igm.ng.Status.FullyLabeledNodes

	if (igm.ng.DeletionTimestamp != nil || igm.ng.Spec.Nodes == 0) && igm.ng.Status.FullyLabeledNodes > 0 {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Autoscaler",
			Message:  fmt.Sprintf("Autoscaler %v  will be delete", igm.ng.Name),
		})
		if !dryRun {
			if err = igm.cm.conn.deleteAutoScalingGroup(igm.ng.Name); err != nil {
				return
			}
		}
		launchConfig := igm.cm.namer.LaunchConfigName(igm.ng.Spec.Template.Spec.SKU)
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Launch configuration",
			Message:  fmt.Sprintf("Launch configuration %v  will be delete", launchConfig),
		})
		if !dryRun {
			if err = igm.cm.conn.deleteLaunchConfiguration(launchConfig); err != nil {
				return
			}
		}
		if !dryRun {
			Store(igm.cm.ctx).NodeGroups(igm.cm.cluster.Name).Delete(igm.ng.Name)
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
			Resource: "Lunch Configuration",
			Message:  fmt.Sprintf("Lunch configuration %v will be created", igm.cm.namer.LaunchConfigName(igm.ng.Spec.Template.Spec.SKU)),
		})

		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Auto scaler",
			Message:  fmt.Sprintf("Autoscaler %v will be created", igm.ng.Name),
		})
		if !dryRun {
			if err = igm.createNodeGroup(igm.ng); err != nil {
				return
			}
			Store(igm.cm.ctx).NodeGroups(igm.cm.cluster.Name).Update(igm.ng)
		}
	} else {
		if adjust < 0 {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "NodeGroup",
				Message:  fmt.Sprintf("%v node will be deleted from %v group", igm.ng.Status.FullyLabeledNodes-igm.ng.Spec.Nodes, igm.ng.Name),
			})
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionAdd,
				Resource: "NodeGroup",
				Message:  fmt.Sprintf("%v node will be added to %v group", igm.ng.Spec.Nodes-igm.ng.Status.FullyLabeledNodes, igm.ng.Name),
			})
		}
		if !dryRun {
			if err = igm.updateNodeGroup(igm.ng, igm.ng.Spec.Nodes); err != nil {
				return
			}
			Store(igm.cm.ctx).NodeGroups(igm.cm.cluster.Name).Update(igm.ng)
		}
	}
	Store(igm.cm.ctx).Clusters().UpdateStatus(igm.cm.cluster)
	return
}

func (igm *AWSNodeGroupManager) createNodeGroup(ng *api.NodeGroup) error {
	launchConfig := igm.cm.namer.LaunchConfigName(ng.Spec.Template.Spec.SKU)

	if err := igm.cm.conn.createLaunchConfiguration(launchConfig, ng); err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if err := igm.cm.conn.createAutoScalingGroup(ng.Name, launchConfig, ng.Spec.Nodes); err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	return nil
}

func (igm *AWSNodeGroupManager) updateNodeGroup(ng *api.NodeGroup, size int64) error {
	group, err := igm.cm.conn.describeGroupInfo(ng.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if size > *group.AutoScalingGroups[0].MaxSize {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: StringP(ng.Name),
			DefaultCooldown:      Int64P(1),
			MaxSize:              Int64P(size),
			DesiredCapacity:      Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}

	} else if size < *group.AutoScalingGroups[0].MinSize {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: StringP(ng.Name),
			MinSize:              Int64P(size),
			DesiredCapacity:      Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	} else {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: StringP(ng.Name),
			DesiredCapacity:      Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}

	//time.Sleep(2 * time.Minute)
	return nil
}
