package eks

import (
	"context"
	"fmt"
	"strings"

	core_util "github.com/appscode/kutil/core/v1"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/ghodss/yaml"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type EKSNodeGroupManager struct {
	ctx  context.Context
	conn *cloudConnector
	ng   *clusterapi.MachineSet
	kc   kubernetes.Interface

	owner string
}

func NewEKSNodeGroupManager(ctx context.Context, conn *cloudConnector, ng *clusterapi.MachineSet, kc kubernetes.Interface, owner string) *EKSNodeGroupManager {
	return &EKSNodeGroupManager{ctx: ctx, conn: conn, ng: ng, kc: kc, owner: owner}
}

func (igm *EKSNodeGroupManager) Apply(dryRun bool) (acts []api.Action, err error) {
	fileName := igm.ng.Name
	igm.ng.Name = strings.Replace(igm.ng.Name, ".", "-", -1)
	//var template []byte
	var found bool
	if found, err = igm.conn.isStackExists(igm.ng.Name); err != nil {
		return
	}
	/*template, err = Asset("amazon-eks-nodegroup.yaml")
	if err != nil {
		return
	}*/

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Node pool",
			Message:  fmt.Sprintf("Node pool %v will be created", igm.ng.Name),
		})
		if !dryRun {
			params := igm.buildstackParams()
			if err = igm.conn.createStack(igm.ng.Name, NodeGroupUrl, params, true); err != nil {
				return
			}
			var ngInfo *cloudformation.Stack
			ngInfo, err = igm.conn.getStack(igm.ng.Name)
			if err != nil {
				return
			}
			if err = igm.newNodeAuthConfigMap(igm.conn.getOutput(ngInfo, "NodeInstanceRole")); err != nil {
				return
			}
		}

	} else {
		if *igm.ng.Spec.Replicas == 0 || igm.ng.DeletionTimestamp != nil {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Node pool",
				Message:  fmt.Sprintf("Node pool %v will be deleted", igm.ng.Name),
			})
			if !dryRun {
				var ngInfo *cloudformation.Stack
				ngInfo, err = igm.conn.getStack(igm.ng.Name)
				if err != nil {
					return
				}
				if err = igm.conn.deleteStack(igm.ng.Name); err != nil {
					return
				}
				if err = igm.deleteNodeAuthConfigMap(igm.conn.getOutput(ngInfo, "NodeInstanceRole")); err != nil {
					return
				}
				err = Store(igm.ctx).Owner(igm.owner).MachineSet(igm.conn.cluster.Name).Delete(fileName)
				if err != nil {
					return acts, err
				}
				return
			}
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionUpdate,
				Resource: "Node pool",
				Message:  fmt.Sprintf("Node pool %v will be updated", igm.ng.Name),
			})
			if !dryRun {
				existingStack, err := igm.conn.getStack(igm.ng.Name)
				if err != nil {
					return acts, err
				}
				params := igm.buildstackParams()
				if err = igm.conn.updateStack(igm.ng.Name, params, true, igm.conn.getOutput(existingStack, "NodeInstanceRole")); err != nil {
					Logger(igm.ctx).Infoln(err)
				}
			}
		}
	}
	igm.ng.Status.Replicas = *igm.ng.Spec.Replicas
	Store(igm.ctx).Owner(igm.owner).MachineSet(igm.conn.cluster.Name).UpdateStatus(igm.ng)

	return acts, err
}

func (igm *EKSNodeGroupManager) buildstackParams() map[string]string {
	providerSpec := igm.conn.cluster.EKSProviderConfig(igm.ng.Spec.Template.Spec.ProviderSpec.Value.Raw)
	return map[string]string{
		"ClusterName":                         igm.conn.cluster.Name,
		"NodeGroupName":                       igm.ng.Name,
		"KeyName":                             igm.conn.cluster.Spec.Config.Cloud.SSHKeyName,
		"NodeImageId":                         igm.conn.cluster.Spec.Config.Cloud.InstanceImage,
		"NodeInstanceType":                    providerSpec.InstanceType,
		"NodeAutoScalingGroupDesiredCapacity": fmt.Sprintf("%d", *igm.ng.Spec.Replicas),
		"NodeAutoScalingGroupMinSize":         fmt.Sprintf("%d", *igm.ng.Spec.Replicas),
		"NodeAutoScalingGroupMaxSize":         fmt.Sprintf("%d", *igm.ng.Spec.Replicas),
		"ClusterControlPlaneSecurityGroup":    igm.conn.cluster.Status.Cloud.EKS.SecurityGroup,
		"Subnets":                             igm.conn.cluster.Status.Cloud.EKS.SubnetId,
		"VpcId":                               igm.conn.cluster.Status.Cloud.EKS.VpcId,
	}
}

func (igm *EKSNodeGroupManager) deleteNodeAuthConfigMap(arn *string) error {
	configmaps, err := igm.kc.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(EKSNodeConfigMap, metav1.GetOptions{})
	if err != nil {
		return err
	}
	mapRoles := make([]map[string]interface{}, 0)
	if configmaps != nil {
		existingRules := configmaps.Data[EKSConfigMapRoles]
		if err := yaml.Unmarshal([]byte(existingRules), &mapRoles); err != nil {
			return err
		}
	}
	newRoles := make([]map[string]interface{}, 0)
	for i, r := range mapRoles {
		if r["rolearn"] != *arn {
			newRoles = append(newRoles, mapRoles[i])
			//delete(mapRoles, r)
		}
	}
	mapRolesBytes, err := yaml.Marshal(newRoles)
	if err != nil {
		return err
	}

	_, _, err = core_util.CreateOrPatchConfigMap(igm.kc,
		metav1.ObjectMeta{Namespace: metav1.NamespaceSystem, Name: EKSNodeConfigMap},
		func(in *core.ConfigMap) *core.ConfigMap {
			if in.Data == nil {
				in.Data = make(map[string]string)
			}
			in.Data[EKSConfigMapRoles] = string(mapRolesBytes)
			return in
		})
	return err
}

func (igm *EKSNodeGroupManager) newNodeAuthConfigMap(arn *string) error {
	mapRoles := make([]map[string]interface{}, 1)
	newRole := make(map[string]interface{})

	newRole["rolearn"] = arn
	newRole["username"] = "system:node:{{EC2PrivateDNSName}}"
	newRole["groups"] = []string{
		"system:bootstrappers",
		"system:nodes",
	}

	configmaps, err := igm.kc.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(EKSNodeConfigMap, metav1.GetOptions{})
	if err == nil {
		existingRules := configmaps.Data[EKSConfigMapRoles]
		if err := yaml.Unmarshal([]byte(existingRules), &mapRoles); err != nil {
			return err
		}
	}
	mapRoles = append(mapRoles, newRole)

	mapRolesBytes, err := yaml.Marshal(mapRoles)
	if err != nil {
		return err
	}

	_, _, err = core_util.CreateOrPatchConfigMap(igm.kc,
		metav1.ObjectMeta{Namespace: metav1.NamespaceSystem, Name: EKSNodeConfigMap},
		func(in *core.ConfigMap) *core.ConfigMap {
			if in.Data == nil {
				in.Data = make(map[string]string)
			}
			in.Data[EKSConfigMapRoles] = string(mapRolesBytes)
			return in
		})
	return err
}
