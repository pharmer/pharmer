package aws

import (
	"fmt"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
)

func (cm *ClusterManager) SetVersion(req *proto.ClusterReconfigureRequest) error {
	if !cloud.UpgradeRequired(cm.cluster, req) {
		cm.ctx.Logger().Infof("Upgrade command skipped for cluster %v", cm.cluster.Name)
		return nil
	}

	if cm.conn == nil {
		conn, err := NewConnector(cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}

	cm.cluster.Spec.ResourceVersion = int64(0)
	cm.namer = namer{cluster: cm.cluster}
	// assign new timestamp and new launch_config version
	cm.cluster.Spec.EnvTimestamp = time.Now().UTC().Format("2006-01-02T15:04:05-0700")
	cm.cluster.Spec.KubernetesVersion = req.KubernetesVersion

	_, err := cm.ctx.Store().Clusters().Update(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	exists, err := cm.findVPC()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if !exists {
		return errors.Newf("VPC %v not found for Cluster %v", cm.cluster.Spec.VpcId, cm.cluster.Name).WithContext(cm.ctx).Err()
	}

	fmt.Println("Updating...")
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances, _ = cm.ctx.Store().Instances(cm.cluster.Name).List(api.ListOptions{})
	if err = cm.conn.detectJessieImage(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if req.ApplyToMaster {
		err = cm.updateMaster()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	} else {
		err = cm.updateNodes(req.Sku)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}

	_, err = cm.ctx.Store().Clusters().Update(cm.cluster)
	cm.ctx.Store().Instances(cm.cluster.Name).SaveInstances(cm.ins.Instances)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Update Completed")
	return nil
}

func (cm *ClusterManager) updateMaster() error {
	if err := cm.deleteMaster(); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	fmt.Println("waiting 1 min")
	time.Sleep(1 * time.Minute)

	if err := cm.restartMaster(); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}
func (cm *ClusterManager) restartMaster() error {
	fmt.Println("Updating Master...")
	cm.UploadStartupConfig()

	masterInstanceID, err := cm.createMasterInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleKubernetesMaster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.waitForInstanceState(masterInstanceID, "running")
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.assignIPToInstance(masterInstanceID)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Logger().Infof("Attaching persistent data volume %v to master", cm.cluster.Spec.MasterDiskId)
	r1, err := cm.conn.ec2.AttachVolume(&_ec2.AttachVolumeInput{
		VolumeId:   types.StringP(cm.cluster.Spec.MasterDiskId),
		Device:     types.StringP("/dev/sdb"),
		InstanceId: types.StringP(masterInstanceID),
	})
	cm.ctx.Logger().Debugln("Attached persistent data volume to master", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err := cloud.ProbeKubeAPI(cm.ctx, cm.cluster); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instance, err := cm.newKubeInstance(masterInstanceID) // sets external IP
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	instance.Spec.Role = api.RoleKubernetesMaster
	// cm.ins.Instances = nil
	// cm.ins.Instances = append(cm.ins.Instances, instance)
	for i := range cm.ins.Instances {
		if cm.ins.Instances[i].Spec.Role == api.RoleKubernetesMaster {
			cm.ins.Instances[i].Status.Phase = api.InstancePhaseDeleted
		}
	}
	cm.ins.Instances = append(cm.ins.Instances, instance)
	fmt.Println("Master updated.")
	return nil
}

func (cm *ClusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")
	/*gc, err := cm.getChanges()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for _, c := range gc {*/
	ctxV, err := cloud.GetExistingContextVersion(cm.cluster, sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	groupName := cm.namer.AutoScalingGroupName(sku)
	cm.ctx.Logger().Infof(" Updating Node groups %v", groupName)
	// TODO: Namer needs fix
	newLaunchConfig := cm.namer.LaunchConfigName(sku)
	oldLaunchConfig := cm.namer.LaunchConfigNameWithContext(sku, ctxV)
	ok, err := cm.LaunchConfigurationExists(newLaunchConfig)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if !ok {
		err = cm.createLaunchConfiguration(newLaunchConfig, sku)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		oldinstances, err := cm.listInstances(groupName)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		instances := []string{}
		for _, instance := range oldinstances {
			instances = append(instances, instance.Status.ExternalID)
		}
		err = cm.rollingUpdate(instances, newLaunchConfig, sku)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		currentIns, err := cm.listInstances(groupName)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cloud.AdjustDbInstance(cm.ctx, cm.ins, currentIns, sku)
		// cluster.Spec.ctx.Instances = append(cluster.Spec.ctx.Instances, instances...)
		_, err = cm.ctx.Store().Clusters().Update(cm.cluster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.deleteLaunchConfiguration(oldLaunchConfig)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	//}
	fmt.Println("Nodes updated.")
	return nil
}

type change struct {
	groupName       string
	sku             string
	desiredCapacity int64
	maxSize         int64
}

func (cm *ClusterManager) getChanges() ([]*change, error) {
	r1, err := cm.conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	changes := make([]*change, 0)
	for _, g := range r1.AutoScalingGroups {
		name := *g.AutoScalingGroupName
		for _, t := range g.Tags {
			if *t.Key == "KubernetesCluster" && *t.Value == cm.cluster.Name {
				changes = append(changes, &change{
					groupName:       name,
					sku:             strings.TrimPrefix(name, cm.cluster.Name+"-node-group-"),
					desiredCapacity: *g.DesiredCapacity,
					maxSize:         *g.MaxSize,
				})
			}
		}
	}
	return changes, nil
}

func (cm *ClusterManager) rollingUpdate(oldInstances []string, newLaunchConfig, sku string) error {
	groupName := cm.namer.AutoScalingGroupName(sku)

	fmt.Println("Updating autoscalling group")
	_, err := cm.conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName:    types.StringP(groupName),
		LaunchConfigurationName: types.StringP(newLaunchConfig),
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	fmt.Println("rolling update started...")

	for _, instance := range oldInstances {
		fmt.Println("updating ", instance)
		_, err = cm.conn.ec2.TerminateInstances(&_ec2.TerminateInstancesInput{
			InstanceIds: []*string{types.StringP(instance)},
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		fmt.Println("Waiting for 1 minute")
		time.Sleep(1 * time.Minute)
		err = cloud.WaitForReadyNodes(cm.ctx, cm.cluster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}

	return nil
}

func (cm *ClusterManager) LaunchConfigurationExists(name string) (bool, error) {
	r, err := cm.conn.autoscale.DescribeLaunchConfigurations(&autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{
			types.StringP(name),
		},
	})
	if err != nil {
		return false, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if len(r.LaunchConfigurations) == 0 {
		return false, nil
	}
	return true, nil
}
