package aws

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

type AWSNodeGroupManager struct {
	cm       *ClusterManager
	instance Instance
}

func (igm *AWSNodeGroupManager) AdjustNodeGroup(dryRun bool) (acts []api.Action, err error) {
	instanceGroupName := igm.cm.namer.AutoScalingGroupName(igm.instance.Type.Sku)
	adjust, _ := Mutator(igm.cm.ctx, igm.cm.cluster, igm.instance, instanceGroupName)
	message := ""
	var action api.ActionType
	if adjust == 0 {
		message = "No changed will be applied"
		action = api.ActionNOP
	} else if adjust < 0 {
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
	if adjust == 0 {
		return
	}
	igm.cm.cluster.Generation = igm.instance.Type.ContextVersion
	if err = igm.cm.conn.detectUbuntuImage(); err != nil {
		return
	}
	if adjust == igm.instance.Stats.Count {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Lunch Configuration",
			Message:  fmt.Sprintf("Lunch configuration %v will be created", igm.cm.namer.LaunchConfigName(igm.instance.Type.Sku)),
		})

		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Auto scaler",
			Message:  fmt.Sprintf("Autoscaler %v will be created", igm.cm.namer.AutoScalingGroupName(igm.instance.Type.Sku)),
		})
		if !dryRun {
			if err = igm.startNodes(igm.instance.Type.Sku, igm.instance.Stats.Count); err != nil {
				return
			}
		}
	} else if igm.instance.Stats.Count == 0 {
		autoscaler := igm.cm.namer.AutoScalingGroupName(igm.instance.Type.Sku)
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Autoscaler",
			Message:  fmt.Sprintf("Autoscaler %v  will be delete", autoscaler),
		})
		if !dryRun {
			if err = igm.cm.conn.deleteAutoScalingGroup(igm.cm.namer.AutoScalingGroupName(igm.instance.Type.Sku)); err != nil {
				return
			}
		}
		launchConfig := igm.cm.namer.LaunchConfigName(igm.instance.Type.Sku)
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
	} else {
		if err = igm.updateNodeGroup(instanceGroupName, igm.instance.Stats.Count); err != nil {
			return
		}
	}
	Store(igm.cm.ctx).Clusters().UpdateStatus(igm.cm.cluster)
	return
}

func (igm *AWSNodeGroupManager) checkNodeGroup(instanceGroup string) (bool, error) {
	groups, err := igm.describeGroupInfo(instanceGroup)
	if err != nil {
		return false, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if len(groups.AutoScalingGroups) > 0 {
		return true, nil
	}
	return false, nil
}

func (igm *AWSNodeGroupManager) startNodes(sku string, count int64) error {
	launchConfig := igm.cm.namer.LaunchConfigName(sku)
	scalingGroup := igm.cm.namer.AutoScalingGroupName(sku)

	err := igm.createLaunchConfiguration(launchConfig, sku)
	err = igm.cm.conn.createAutoScalingGroup(scalingGroup, launchConfig, count)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	return nil
}

func (igm *AWSNodeGroupManager) createLaunchConfiguration(name, sku string) error {
	//script := igm.cm.RenderStartupScript(igm.cm.ctx.NewScriptOptions(), sku, system.RoleKubernetesPool)
	script, err := RenderStartupScript(igm.cm.ctx, igm.cm.cluster, api.RoleNode, igm.cm.namer.AutoScalingGroupName(igm.instance.Type.Sku))
	if err != nil {
		return err
	}

	Logger(igm.cm.ctx).Info("Creating node configuration assuming enableNodePublicIP = true")
	fmt.Println(igm.cm.cluster.Status.Cloud.AWS.RootDeviceName, "<<<<<<<<--------------->>>>>>>>>>>>>>>>>>.")
	configuration := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  StringP(name),
		AssociatePublicIpAddress: BoolP(igm.cm.cluster.Spec.EnableNodePublicIP),
		/*
			// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html
			BlockDeviceMappings: []*autoscaling.BlockDeviceMapping{
				// NODE_BLOCK_DEVICE_MAPPINGS
				{
					// https://github.com/appscode/kubernetes/blob/55d9dec8eb5eb02e1301045b7b81bbac689c86a1/cluster/aws/util.sh#L397
					DeviceName: StringP(igm.cm.cluster.Spec.RootDeviceName),
					Ebs: &autoscaling.Ebs{
						DeleteOnTermination: TrueP(),
						VolumeSize:          Int64P(igm.cm.conn.cluster.Spec.NodeDiskSize),
						VolumeType:          StringP(igm.cm.cluster.Spec.NodeDiskType),
					},
				},
				// EPHEMERAL_BLOCK_DEVICE_MAPPINGS
				{
					DeviceName:  StringP("/dev/sdc"),
					VirtualName: StringP("ephemeral0"),
				},
				{
					DeviceName:  StringP("/dev/sdd"),
					VirtualName: StringP("ephemeral1"),
				},
				{
					DeviceName:  StringP("/dev/sde"),
					VirtualName: StringP("ephemeral2"),
				},
				{
					DeviceName:  StringP("/dev/sdf"),
					VirtualName: StringP("ephemeral3"),
				},
			},
		*/
		IamInstanceProfile: StringP(igm.cm.cluster.Spec.Cloud.AWS.IAMProfileNode),
		ImageId:            StringP(igm.cm.cluster.Spec.Cloud.InstanceImage),
		InstanceType:       StringP(sku),
		KeyName:            StringP(igm.cm.cluster.Status.SSHKeyExternalID),
		SecurityGroups: []*string{
			StringP(igm.cm.cluster.Status.Cloud.AWS.NodeSGId),
		},
		UserData: StringP(base64.StdEncoding.EncodeToString([]byte(script))),
	}
	r1, err := igm.cm.conn.autoscale.CreateLaunchConfiguration(configuration)
	Logger(igm.cm.ctx).Debug("Created node configuration", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	return nil
}

func (igm *AWSNodeGroupManager) deleteOnlyNodeGroup(instanceGroup string) error {
	_, err := igm.describeGroupInfo(instanceGroup)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	return nil
}

func (igm *AWSNodeGroupManager) updateNodeGroup(instanceGroup string, size int64) error {
	group, err := igm.describeGroupInfo(instanceGroup)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if size > *group.AutoScalingGroups[0].MaxSize {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: StringP(instanceGroup),
			DefaultCooldown:      Int64P(1),
			MaxSize:              Int64P(size),
			DesiredCapacity:      Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}

	} else if size < *group.AutoScalingGroups[0].MinSize {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: StringP(instanceGroup),
			MinSize:              Int64P(size),
			DesiredCapacity:      Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	} else {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: StringP(instanceGroup),
			DesiredCapacity:      Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	time.Sleep(2 * time.Minute)
	return nil
}

func (igm *AWSNodeGroupManager) listInstances(instanceGroup string) ([]*api.Node, error) {
	Logger(igm.cm.ctx).Infof("Retrieving instances in node group %v", instanceGroup)
	instances := make([]*api.Node, 0)
	group, err := igm.describeGroupInfo(instanceGroup)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	for _, item := range group.AutoScalingGroups[0].Instances {
		instance, err := igm.cm.conn.newKubeInstance(*item.InstanceId)
		instance.Spec.Role = api.RoleNode
		instances = append(instances, instance)
		if err != nil {
			return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	return instances, nil
}

func (igm *AWSNodeGroupManager) describeGroupInfo(instanceGroup string) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	groups := make([]*string, 0)
	groups = append(groups, StringP(instanceGroup))
	r1, err := igm.cm.conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groups,
	})
	if err != nil {
		return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	return r1, nil
}
