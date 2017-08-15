package aws

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/system"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

type InstanceGroupManager struct {
	cm       *clusterManager
	instance lib.Instance
}

func (igm *InstanceGroupManager) AdjustInstanceGroup() error {
	instanceGroupName := igm.cm.namer.AutoScalingGroupName(igm.instance.Type.Sku)
	found, err := igm.checkInstanceGroup(instanceGroupName)
	if err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	igm.cm.ctx.ContextVersion = igm.instance.Type.ContextVersion
	igm.cm.ctx.Load()
	if err = igm.cm.conn.detectJessieImage(); err != nil {
		igm.cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if !found {
		if err := igm.startNodes(igm.instance.Type.Sku, igm.instance.Stats.Count); err != nil {
			igm.cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithMessage("failed to start node").WithContext(igm.cm.ctx).Err()
		}
	} else if igm.instance.Stats.Count == 0 {
		err = igm.deleteOnlyInstanceGroup(instanceGroupName)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}

		err = igm.cm.deleteAutoScalingGroup(igm.cm.namer.AutoScalingGroupName(igm.instance.Type.Sku))
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	} else {
		err := igm.updateInstanceGroup(instanceGroupName, igm.instance.Stats.Count)
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	igm.cm.ctx.Save()
	return nil
}

func (igm *InstanceGroupManager) checkInstanceGroup(instanceGroup string) (bool, error) {
	groups, err := igm.describeGroupInfo(instanceGroup)
	if err != nil {
		return false, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if len(groups.AutoScalingGroups) > 0 {
		return true, nil
	}
	return false, nil
}

func (igm *InstanceGroupManager) startNodes(sku string, count int64) error {
	launchConfig := igm.cm.namer.LaunchConfigName(sku)
	scalingGroup := igm.cm.namer.AutoScalingGroupName(sku)

	err := igm.createLaunchConfiguration(launchConfig, sku)
	err = igm.cm.createAutoScalingGroup(scalingGroup, launchConfig, count)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	return nil
}

func (igm *InstanceGroupManager) createLaunchConfiguration(name, sku string) error {
	script := igm.cm.RenderStartupScript(igm.cm.ctx.NewScriptOptions(), sku, system.RoleKubernetesPool)

	igm.cm.ctx.Logger().Info("Creating node configuration assuming EnableNodePublicIP = true")
	fmt.Println(igm.cm.ctx.RootDeviceName, "<<<<<<<<--------------->>>>>>>>>>>>>>>>>>.")
	configuration := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  types.StringP(name),
		AssociatePublicIpAddress: types.BoolP(igm.cm.ctx.EnableNodePublicIP),
		// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html
		BlockDeviceMappings: []*autoscaling.BlockDeviceMapping{
			// NODE_BLOCK_DEVICE_MAPPINGS
			{
				// https://github.com/appscode/kubernetes/blob/55d9dec8eb5eb02e1301045b7b81bbac689c86a1/cluster/aws/util.sh#L397
				DeviceName: types.StringP(igm.cm.ctx.RootDeviceName),
				Ebs: &autoscaling.Ebs{
					DeleteOnTermination: types.TrueP(),
					VolumeSize:          types.Int64P(igm.cm.conn.ctx.NodeDiskSize),
					VolumeType:          types.StringP(igm.cm.ctx.NodeDiskType),
				},
			},
			// EPHEMERAL_BLOCK_DEVICE_MAPPINGS
			{
				DeviceName:  types.StringP("/dev/sdc"),
				VirtualName: types.StringP("ephemeral0"),
			},
			{
				DeviceName:  types.StringP("/dev/sdd"),
				VirtualName: types.StringP("ephemeral1"),
			},
			{
				DeviceName:  types.StringP("/dev/sde"),
				VirtualName: types.StringP("ephemeral2"),
			},
			{
				DeviceName:  types.StringP("/dev/sdf"),
				VirtualName: types.StringP("ephemeral3"),
			},
		},
		IamInstanceProfile: types.StringP(igm.cm.ctx.IAMProfileNode),
		ImageId:            types.StringP(igm.cm.ctx.InstanceImage),
		InstanceType:       types.StringP(sku),
		KeyName:            types.StringP(igm.cm.ctx.SSHKeyExternalID),
		SecurityGroups: []*string{
			types.StringP(igm.cm.ctx.NodeSGId),
		},
		UserData: types.StringP(base64.StdEncoding.EncodeToString([]byte(script))),
	}
	r1, err := igm.cm.conn.autoscale.CreateLaunchConfiguration(configuration)
	igm.cm.ctx.Logger().Debug("Created node configuration", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	return nil
}

func (igm *InstanceGroupManager) deleteOnlyInstanceGroup(instanceGroup string) error {
	_, err := igm.describeGroupInfo(instanceGroup)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	return nil
}

func (igm *InstanceGroupManager) updateInstanceGroup(instanceGroup string, size int64) error {
	group, err := igm.describeGroupInfo(instanceGroup)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	if size > *group.AutoScalingGroups[0].MaxSize {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: types.StringP(instanceGroup),
			DefaultCooldown:      types.Int64P(1),
			MaxSize:              types.Int64P(size),
			DesiredCapacity:      types.Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}

	} else if size < *group.AutoScalingGroups[0].MinSize {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: types.StringP(instanceGroup),
			MinSize:              types.Int64P(size),
			DesiredCapacity:      types.Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	} else {
		_, err := igm.cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: types.StringP(instanceGroup),
			DesiredCapacity:      types.Int64P(size),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	sz := *group.AutoScalingGroups[0].DesiredCapacity
	fmt.Println("Updating autoscaling group...")
	time.Sleep(2 * time.Minute)
	err = lib.WaitForReadyNodes(igm.cm.ctx, size-sz)
	if err != nil {
		return errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	return nil
}

func (igm *InstanceGroupManager) listInstances(instanceGroup string) ([]*api.KubernetesInstance, error) {
	igm.cm.ctx.Logger().Infof("Retrieving instances in node group %v", instanceGroup)
	instances := make([]*api.KubernetesInstance, 0)
	group, err := igm.describeGroupInfo(instanceGroup)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}

	for _, item := range group.AutoScalingGroups[0].Instances {
		instance, err := igm.cm.newKubeInstance(*item.InstanceId)
		instance.Role = system.RoleKubernetesPool
		instances = append(instances, instance)
		if err != nil {
			return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
		}
	}
	return instances, nil
}

func (igm *InstanceGroupManager) describeGroupInfo(instanceGroup string) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	groups := make([]*string, 0)
	groups = append(groups, types.StringP(instanceGroup))
	r1, err := igm.cm.conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groups,
	})
	if err != nil {
		return nil, errors.FromErr(err).WithContext(igm.cm.ctx).Err()
	}
	return r1, nil
}
