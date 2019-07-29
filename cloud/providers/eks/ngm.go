package eks

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/ghodss/yaml"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	"pharmer.dev/pharmer/apis/v1alpha1/aws"
	"pharmer.dev/pharmer/cloud"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type EKSNodeGroupManager struct {
	*cloud.Scope
	conn *cloudConnector
	ng   *clusterapi.MachineSet
	kc   kubernetes.Interface
}

func NewEKSNodeGroupManager(scope *cloud.Scope, conn *cloudConnector, ng *clusterapi.MachineSet, kc kubernetes.Interface) *EKSNodeGroupManager {
	return &EKSNodeGroupManager{Scope: scope, conn: conn, ng: ng, kc: kc}
}

func (igm *EKSNodeGroupManager) Apply() error {
	log := igm.Logger
	fileName := igm.ng.Name
	igm.ng.Name = strings.Replace(igm.ng.Name, ".", "-", -1)

	var found bool
	var err error

	found = igm.conn.isStackExists(igm.ng.Name)

	if !found {
		params, err := igm.buildstackParams()
		if err != nil {
			log.Error(err, "failed to build stack params")
			return err
		}
		if err = igm.conn.createStack(igm.ng.Name, NodeGroupURL, params, true); err != nil {
			log.Error(err, "failed to node group")
			return err
		}
		var ngInfo *cloudformation.Stack
		ngInfo, err = igm.conn.getStack(igm.ng.Name)
		if err != nil {
			log.Error(err, "failed to get node group information")
			return err
		}
		if err = igm.newNodeAuthConfigMap(igm.conn.getOutput(ngInfo, "NodeInstanceRole")); err != nil {
			log.Error(err, "failed to get node auth configmap")
			return err
		}
	} else {
		if *igm.ng.Spec.Replicas == 0 || igm.ng.DeletionTimestamp != nil {
			if err = igm.conn.deleteStack(igm.ng.Name); err != nil {
				log.Error(err, "failed to delete stack")
				return err
			}
			err = igm.StoreProvider.MachineSet(igm.conn.Cluster.Name).Delete(fileName)
			if err != nil {
				log.Error(err, "failed to delete machineset from store")
				return err
			}

			return nil
		} else {
			params, err := igm.buildstackParams()
			if err != nil {
				log.Error(err, "failed to build stack params")
				return err
			}
			if err = igm.conn.updateStack(igm.ng.Name, params, true); err != nil {
				igm.conn.Logger.Error(err, "error updating stack")
			}
		}
	}

	igm.ng.Name = fileName
	igm.ng.Status.Replicas = *igm.ng.Spec.Replicas
	_, err = igm.StoreProvider.MachineSet(igm.conn.Cluster.Name).UpdateStatus(igm.ng)
	if err != nil {
		log.Error(err, "failed to update machine set status in store")
		return err
	}

	return nil
}

func (igm *EKSNodeGroupManager) buildstackParams() (map[string]string, error) {
	providerSpec, err := aws.MachineConfigFromProviderSpec(igm.ng.Spec.Template.Spec.ProviderSpec)
	if err != nil {
		igm.conn.Logger.Error(err, "error getting providerspec")
		return nil, err
	}
	return map[string]string{
		"ClusterName":                         igm.conn.Cluster.Name,
		"NodeGroupName":                       igm.ng.Name,
		"KeyName":                             igm.conn.Cluster.Spec.Config.Cloud.SSHKeyName,
		"NodeImageId":                         igm.conn.Cluster.Spec.Config.Cloud.InstanceImage,
		"NodeInstanceType":                    providerSpec.InstanceType,
		"NodeAutoScalingGroupDesiredCapacity": fmt.Sprintf("%d", *igm.ng.Spec.Replicas),
		"NodeAutoScalingGroupMinSize":         fmt.Sprintf("%d", *igm.ng.Spec.Replicas),
		"NodeAutoScalingGroupMaxSize":         fmt.Sprintf("%d", *igm.ng.Spec.Replicas),
		"ClusterControlPlaneSecurityGroup":    igm.conn.Cluster.Status.Cloud.EKS.SecurityGroup,
		"Subnets":                             igm.conn.Cluster.Status.Cloud.EKS.SubnetID,
		"VpcId":                               igm.conn.Cluster.Status.Cloud.EKS.VpcID,
	}, nil
}

func (igm *EKSNodeGroupManager) newNodeAuthConfigMap(arn *string) error {
	log := igm.Logger
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
			log.Error(err, "failed to unmarshal existing rules yaml")
			return err
		}
	}
	mapRoles = append(mapRoles, newRole)

	mapRolesBytes, err := yaml.Marshal(mapRoles)
	if err != nil {
		log.Error(err, "failed to marshal new roles yaml")
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
	if err != nil {
		log.Error(err, "failed to create configmap")
		return err
	}
	return nil
}
