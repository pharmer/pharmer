package aws

import (
	"fmt"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/data"
	"github.com/appscode/errors"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
)

func (cm *clusterManager) setVersion(req *proto.ClusterReconfigureRequest) error {
	if !lib.UpgradeRequired(cm.ctx, req) {
		cm.ctx.Logger().Warningf("Upgrade command skipped for cluster %v", cm.ctx.Name)
		return nil
	}

	if cm.conn == nil {
		conn, err := NewConnector(cm.ctx)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}

	cm.ctx.ContextVersion = int64(0)
	cm.namer = namer{ctx: cm.ctx}
	// assign new timestamp and new launch_config version
	cm.ctx.EnvTimestamp = time.Now().UTC().Format("2006-01-02T15:04:05-0700")
	if req.Version != "" {
		if v, err := data.LoadKubernetesVersion(cm.ctx.Provider, _env.FromHost().String(), req.Version); err == nil {
			cm.ctx.SaltbaseVersion = v.Apps[system.AppKubeSaltbase]
			cm.ctx.KubeServerVersion = v.Apps[system.AppKubeServer]
			cm.ctx.KubeStarterVersion = v.Apps[system.AppKubeStarter]
			cm.ctx.HostfactsVersion = v.Apps[system.AppHostfacts]
		}
	}
	cm.ctx.KubeVersion = req.Version
	cm.ctx.Apps[system.AppKubeSaltbase] = system.NewAppKubernetesSalt(cm.ctx.Provider, cm.ctx.Region, cm.ctx.SaltbaseVersion)
	cm.ctx.Apps[system.AppKubeServer] = system.NewAppKubernetesServer(cm.ctx.Provider, cm.ctx.Region, cm.ctx.KubeServerVersion)
	cm.ctx.Apps[system.AppKubeStarter] = system.NewAppStartKubernetes(cm.ctx.Provider, cm.ctx.Region, cm.ctx.KubeStarterVersion)

	err := cm.ctx.Save()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	exists, err := cm.findVPC()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if !exists {
		return errors.Newf("VPC %v not found for Cluster %v", cm.ctx.VpcId, cm.ctx.Name).WithContext(cm.ctx).Err()
	}

	fmt.Println("Updating...")
	cm.ins, err = lib.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Load()
	if err = cm.conn.detectJessieImage(); err != nil {
		cm.ctx.StatusCause = err.Error()
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

	err = cm.ctx.Save()
	cm.ins.Save()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Update Completed")
	return nil
}

func (cm *clusterManager) updateMaster() error {
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
func (cm *clusterManager) restartMaster() error {
	fmt.Println("Updating Master...")
	cm.UploadStartupConfig()

	masterInstanceID, err := cm.createMasterInstance(cm.ctx.KubernetesMasterName, system.RoleKubernetesMaster)
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

	cm.ctx.Logger().Infof("Attaching persistent data volume %v to master", cm.ctx.MasterDiskId)
	r1, err := cm.conn.ec2.AttachVolume(&_ec2.AttachVolumeInput{
		VolumeId:   types.StringP(cm.ctx.MasterDiskId),
		Device:     types.StringP("/dev/sdb"),
		InstanceId: types.StringP(masterInstanceID),
	})
	cm.ctx.Logger().Debugln("Attached persistent data volume to master", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err := lib.ProbeKubeAPI(cm.ctx); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instance, err := cm.newKubeInstance(masterInstanceID) // sets external IP
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	instance.Role = system.RoleKubernetesMaster
	// cm.ins.Instances = nil
	// cm.ins.Instances = append(cm.ins.Instances, instance)
	for i := range cm.ins.Instances {
		if cm.ins.Instances[i].Role == system.RoleKubernetesMaster {
			cm.ins.Instances[i].Status = storage.KubernetesInstanceStatus_Deleted
		}
	}
	cm.ins.Instances = append(cm.ins.Instances, instance)
	fmt.Println("Master updated.")
	return nil
}

func (cm *clusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")
	/*gc, err := cm.getChanges()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for _, c := range gc {*/
	ctxV, err := lib.GetExistingContextVersion(cm.ctx, sku)
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
			instances = append(instances, instance.ExternalID)
		}
		err = cm.rollingUpdate(instances, newLaunchConfig, sku)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		currentIns, err := cm.listInstances(groupName)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = lib.AdjustDbInstance(cm.ins, currentIns, sku)
		// cluster.ctx.Instances = append(cluster.ctx.Instances, instances...)
		err = cm.ctx.Save()
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

func (cm *clusterManager) getChanges() ([]*change, error) {
	r1, err := cm.conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	changes := make([]*change, 0)
	for _, g := range r1.AutoScalingGroups {
		name := *g.AutoScalingGroupName
		for _, t := range g.Tags {
			if *t.Key == "KubernetesCluster" && *t.Value == cm.ctx.Name {
				changes = append(changes, &change{
					groupName:       name,
					sku:             strings.TrimPrefix(name, cm.ctx.Name+"-node-group-"),
					desiredCapacity: *g.DesiredCapacity,
					maxSize:         *g.MaxSize,
				})
			}
		}
	}
	return changes, nil
}

func (cm *clusterManager) rollingUpdate(oldInstances []string, newLaunchConfig, sku string) error {
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
		err = lib.WaitForReadyNodes(cm.ctx)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}

	return nil
}

func (cm *clusterManager) LaunchConfigurationExists(name string) (bool, error) {
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
