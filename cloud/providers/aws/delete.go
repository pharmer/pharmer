package aws

import (
	"fmt"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/cenkalti/backoff"
)

func (cm *ClusterManager) Delete(req *proto.ClusterDeleteRequest) error {
	defer cm.cluster.Delete()

	if cm.cluster.Status.Phase == api.ClusterPhasePending {
		cm.cluster.Status.Phase = api.ClusterPhaseFailing
	} else if cm.cluster.Status.Phase == api.ClusterPhaseReady {
		cm.cluster.Status.Phase = api.ClusterPhaseDeleting
	}
	// cm.ctx.Store().UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

	if cm.conn == nil {
		conn, err := NewConnector(cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}
	cm.namer = namer{cluster: cm.cluster}

	exists, err := cm.findVPC()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if !exists {
		return errors.Newf("VPC %v not found for Cluster %v", cm.cluster.Spec.VpcId, cm.cluster.Name).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.cluster.Status.Reason != "" {
		errs = append(errs, cm.cluster.Status.Reason)
	}

	for _, ng := range cm.cluster.Spec.NodeGroups {
		if err = cm.deleteAutoScalingGroup(cm.namer.AutoScalingGroupName(ng.Sku)); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if err = cm.deleteMaster(); err != nil {
		errs = append(errs, err.Error())
	}
	if err = cm.ensureInstancesDeleted(); err != nil {
		errs = append(errs, err.Error())
	}
	for _, ng := range cm.cluster.Spec.NodeGroups {
		if err = cm.deleteLaunchConfiguration(cm.namer.AutoScalingGroupName(ng.Sku)); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if err = cm.deleteVolume(); err != nil {
		errs = append(errs, err.Error())
	}

	if req.ReleaseReservedIp && cm.cluster.Spec.MasterReservedIP != "" {
		if err = cm.releaseReservedIP(cm.cluster.Spec.MasterReservedIP); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if err := backoff.Retry(cm.deleteSecurityGroup, backoff.NewExponentialBackOff()); err != nil {
		errs = append(errs, err.Error())
	}

	if err := backoff.Retry(cm.deleteSecurityGroup, backoff.NewExponentialBackOff()); err != nil {
		errs = append(errs, err.Error())
	}

	if err := backoff.Retry(cm.deleteInternetGateway, backoff.NewExponentialBackOff()); err != nil {
		errs = append(errs, err.Error())
	}

	if err := backoff.Retry(cm.deleteDHCPOption, backoff.NewExponentialBackOff()); err != nil {
		errs = append(errs, err.Error())
	}

	if err := backoff.Retry(cm.deleteRouteTable, backoff.NewExponentialBackOff()); err != nil {
		errs = append(errs, err.Error())
	}

	if err := backoff.Retry(cm.deleteSubnetId, backoff.NewExponentialBackOff()); err != nil {
		errs = append(errs, err.Error())
	}

	if err := backoff.Retry(cm.deleteVpc, backoff.NewExponentialBackOff()); err != nil {
		errs = append(errs, err.Error())
	}

	if err = cm.deleteBucket(); err != nil {
		errs = append(errs, err.Error())
	}

	// Delete SSH key from DB
	if err = cm.deleteSSHKey(); err != nil {
		errs = append(errs, err.Error())
	}

	if err = cloud.DeleteARecords(cm.ctx, cm.cluster); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.cluster.Status.Phase == api.ClusterPhaseDeleting {
			cm.cluster.Status.Reason = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	cm.ctx.Logger().Infof("Cluster %v deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) findVPC() (bool, error) {
	r1, err := cm.conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
		VpcIds: []*string{
			types.StringP(cm.cluster.Spec.VpcId),
		},
	})
	if err != nil {
		return false, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return len(r1.Vpcs) > 0, nil
}

func (cm *ClusterManager) deleteAutoScalingGroup(name string) error {
	_, err := cm.conn.autoscale.DeleteAutoScalingGroup(&autoscaling.DeleteAutoScalingGroupInput{
		ForceDelete:          types.TrueP(),
		AutoScalingGroupName: types.StringP(name),
	})
	cm.ctx.Logger().Infof("Auto scaling group %v is deleted for cluster %v", name, cm.cluster.Name)
	return err
}

func (cm *ClusterManager) deleteLaunchConfiguration(name string) error {
	_, err := cm.conn.autoscale.DeleteLaunchConfiguration(&autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: types.StringP(name),
	})
	cm.ctx.Logger().Infof("Launch configuration %v os de;eted for cluster %v", name, cm.cluster.Name)
	return err
}

func (cm *ClusterManager) deleteMaster() error {
	r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:Role"),
				Values: []*string{
					types.StringP(api.RoleKubernetesMaster),
				},
			},
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterInstances := make([]*string, 0)
	for _, reservation := range r1.Reservations {
		for _, instance := range reservation.Instances {
			masterInstances = append(masterInstances, instance.InstanceId)
		}
	}
	fmt.Printf("TerminateInstances %v", stringutil.Join(masterInstances, ","))
	cm.ctx.Logger().Infof("Terminating master instance for cluster %v", cm.cluster.Name)
	_, err = cm.conn.ec2.TerminateInstances(&_ec2.TerminateInstancesInput{
		InstanceIds: masterInstances,
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instanceInput := &_ec2.DescribeInstancesInput{
		InstanceIds: masterInstances,
	}
	err = cm.conn.ec2.WaitUntilInstanceTerminated(instanceInput)
	fmt.Println(err, "--------------------<<<<<<<")
	cm.ctx.Logger().Infof("Master instance for cluster %v is terminated", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) ensureInstancesDeleted() error {
	const desiredState = "terminated"

	r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances := make(map[string]bool)
	for _, reservation := range r1.Reservations {
		for _, instance := range reservation.Instances {
			if *instance.State.Name != desiredState {
				instances[*instance.InstanceId] = true
			}
		}
	}

	for {
		ris := make([]*string, 0)
		for instance, running := range instances {
			if running {
				ris = append(ris, types.StringP(instance))
			}
		}
		fmt.Println("Waiting for instances to terminate", stringutil.Join(ris, ","))

		r2, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{InstanceIds: ris})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		stillRunning := false
		for _, reservation := range r2.Reservations {
			for _, instance := range reservation.Instances {
				if *instance.State.Name == desiredState {
					instances[*instance.InstanceId] = false
				} else {
					stillRunning = true
					instances[*instance.InstanceId] = true
				}
			}
		}

		if !stillRunning {
			break
		}
		time.Sleep(15 * time.Second)
	}
	return nil
}

func (cm *ClusterManager) deleteSecurityGroup() error {
	r, err := cm.conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cm.cluster.Spec.VpcId),
				},
			},
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	for _, sg := range r.SecurityGroups {
		if len(sg.IpPermissions) > 0 {
			_, err := cm.conn.ec2.RevokeSecurityGroupIngress(&_ec2.RevokeSecurityGroupIngressInput{
				GroupId:       sg.GroupId,
				IpPermissions: sg.IpPermissions,
			})
			if err != nil {
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
		}

		if len(sg.IpPermissionsEgress) > 0 {
			_, err := cm.conn.ec2.RevokeSecurityGroupEgress(&_ec2.RevokeSecurityGroupEgressInput{
				GroupId:       sg.GroupId,
				IpPermissions: sg.IpPermissionsEgress,
			})
			if err != nil {
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
		}
	}

	for _, sg := range r.SecurityGroups {
		_, err := cm.conn.ec2.DeleteSecurityGroup(&_ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.ctx.Logger().Infof("Security groups for cluster %v is deleted", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteSubnetId() error {
	r, err := cm.conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cm.cluster.Spec.VpcId),
				},
			},
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for _, subnet := range r.Subnets {
		_, err := cm.conn.ec2.DeleteSubnet(&_ec2.DeleteSubnetInput{
			SubnetId: subnet.SubnetId,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Infof("Subnet ID in VPC %v is deleted", *subnet.SubnetId)
	}
	return nil
}

func (cm *ClusterManager) deleteInternetGateway() error {
	r1, err := cm.conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("attachment.vpc-id"),
				Values: []*string{
					types.StringP(cm.cluster.Spec.VpcId),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for _, igw := range r1.InternetGateways {
		_, err := cm.conn.ec2.DetachInternetGateway(&_ec2.DetachInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
			VpcId:             types.StringP(cm.cluster.Spec.VpcId),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		_, err = cm.conn.ec2.DeleteInternetGateway(&_ec2.DeleteInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.ctx.Logger().Infof("Internet gateway for cluster %v are deleted", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteRouteTable() error {
	r1, err := cm.conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cm.cluster.Spec.VpcId),
				},
			},
		},
	})

	if err != nil {
		fmt.Println(err)
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for _, rt := range r1.RouteTables {
		mainTable := false
		for _, assoc := range rt.Associations {
			if _aws.BoolValue(assoc.Main) {
				mainTable = true
			} else {
				_, err := cm.conn.ec2.DisassociateRouteTable(&_ec2.DisassociateRouteTableInput{
					AssociationId: assoc.RouteTableAssociationId,
				})
				if err != nil {
					fmt.Println(err)
					return errors.FromErr(err).WithContext(cm.ctx).Err()
				}
			}
		}
		if !mainTable {
			_, err := cm.conn.ec2.DeleteRouteTable(&_ec2.DeleteRouteTableInput{
				RouteTableId: rt.RouteTableId,
			})
			if err != nil {
				fmt.Println(err)
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
		}
	}
	cm.ctx.Logger().Infof("Route tables for cluster %v are deleted", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteDHCPOption() error {
	_, err := cm.conn.ec2.AssociateDhcpOptions(&_ec2.AssociateDhcpOptionsInput{
		VpcId:         types.StringP(cm.cluster.Spec.VpcId),
		DhcpOptionsId: types.StringP("default"),
	})
	if err != nil {
		fmt.Println(err)
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	r, err := cm.conn.ec2.DescribeDhcpOptions(&_ec2.DescribeDhcpOptionsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.cluster.Name),
				},
			},
		},
	})
	for _, dhcp := range r.DhcpOptions {
		_, err = cm.conn.ec2.DeleteDhcpOptions(&_ec2.DeleteDhcpOptionsInput{
			DhcpOptionsId: dhcp.DhcpOptionsId,
		})
		if err != nil {
			fmt.Println(err)
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.ctx.Logger().Infof("DHCP options for cluster %v are deleted", cm.cluster.Name)
	return err
}

func (cm *ClusterManager) deleteVpc() error {
	_, err := cm.conn.ec2.DeleteVpc(&_ec2.DeleteVpcInput{
		VpcId: types.StringP(cm.cluster.Spec.VpcId),
	})

	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("VPC for cluster %v is deleted", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteVolume() error {
	_, err := cm.conn.ec2.DeleteVolume(&_ec2.DeleteVolumeInput{
		VolumeId: types.StringP(cm.cluster.Spec.MasterDiskId),
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Master instance volume for cluster %v is deleted", cm.cluster.Spec.MasterDiskId)
	return nil
}

func (cm *ClusterManager) deleteSSHKey() error {
	var err error
	_, err = cm.conn.ec2.DeleteKeyPair(&_ec2.DeleteKeyPairInput{
		KeyName: types.StringP(cm.cluster.Spec.SSHKeyExternalID),
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("SSH key for cluster %v is deleted", cm.cluster.Spec.MasterDiskId)
	//updates := &storage.SSHKey{IsDeleted: 1}
	//cond := &storage.SSHKey{PHID: cluster.Spec.ctx.SSHKeyPHID}
	// _, err = cluster.Spec.ctx.Store().Engine.Update(updates, cond)

	return err
}

func (cm *ClusterManager) releaseReservedIP(publicIP string) error {
	r1, err := cm.conn.ec2.DescribeAddresses(&_ec2.DescribeAddressesInput{
		PublicIps: []*string{
			types.StringP(publicIP),
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	_, err = cm.conn.ec2.ReleaseAddress(&_ec2.ReleaseAddressInput{
		AllocationId: r1.Addresses[0].AllocationId,
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Elastic IP for cluster %v is deleted", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteNetworkInterface(vpcId string) error {
	r, err := cm.conn.ec2.DescribeNetworkInterfaces(&_ec2.DescribeNetworkInterfacesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(vpcId),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	for _, iface := range r.NetworkInterfaces {
		_, err = cm.conn.ec2.DetachNetworkInterface(&_ec2.DetachNetworkInterfaceInput{
			AttachmentId: iface.Attachment.AttachmentId,
			Force:        types.TrueP(),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		time.Sleep(1 * time.Minute)
		_, err = cm.conn.ec2.DeleteNetworkInterface(&_ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: iface.NetworkInterfaceId,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.ctx.Logger().Infof("Network interfaces for cluster %v are deleted", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteBucket() error {
	// http://docs.aws.amazon.com/AmazonS3/latest/dev/delete-or-empty-bucket.html#delete-bucket-awscli
	cm.ctx.Logger().Infof("Deleting startupconfig bucket for cluster %v", cm.cluster.Name)
	var timeout int64 = 30 * 60 // Give max 30 min to empty the bucket
	start := time.Now().Unix()

	for {
		r1, err := cm.conn.s3.ListObjectsV2(&_s3.ListObjectsV2Input{
			Bucket: types.StringP(cm.cluster.Spec.BucketName),
		})
		if err == nil && len(r1.Contents) > 0 {
			oIds := make([]*_s3.ObjectIdentifier, len(r1.Contents))
			for i, obj := range r1.Contents {
				oIds[i] = &_s3.ObjectIdentifier{Key: obj.Key}
			}
			cm.conn.s3.DeleteObjects(&_s3.DeleteObjectsInput{
				Bucket: types.StringP(cm.cluster.Spec.BucketName),
				Delete: &_s3.Delete{Objects: oIds},
			})
		}
		if len(r1.Contents) == 0 || (time.Now().Unix() > start+timeout) {
			break
		}
	}

	for {
		r1, err := cm.conn.s3.ListObjectVersions(&_s3.ListObjectVersionsInput{
			Bucket: types.StringP(cm.cluster.Spec.BucketName),
		})
		if err == nil && len(r1.DeleteMarkers) > 0 {
			oIds := make([]*_s3.ObjectIdentifier, len(r1.DeleteMarkers))
			for i, obj := range r1.DeleteMarkers {
				oIds[i] = &_s3.ObjectIdentifier{Key: obj.Key, VersionId: obj.VersionId}
			}
			cm.conn.s3.DeleteObjects(&_s3.DeleteObjectsInput{
				Bucket: types.StringP(cm.cluster.Spec.BucketName),
				Delete: &_s3.Delete{Objects: oIds},
			})
		}
		if len(r1.DeleteMarkers) == 0 || (time.Now().Unix() > start+timeout) {
			break
		}
	}

	_, err := cm.conn.s3.DeleteBucket(&_s3.DeleteBucketInput{
		Bucket: types.StringP(cm.cluster.Spec.BucketName),
	})
	return err
}
