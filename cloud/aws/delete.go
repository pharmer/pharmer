package aws

import (
	"fmt"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	stringutil "github.com/appscode/go/strings"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/errorhandlers"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/cenkalti/backoff"
)

func (cm *clusterManager) delete(req *proto.ClusterDeleteRequest) error {
	defer cm.ctx.Delete()

	if cm.ctx.Status == storage.KubernetesStatus_Pending {
		cm.ctx.Status = storage.KubernetesStatus_Failing
	} else if cm.ctx.Status == storage.KubernetesStatus_Ready {
		cm.ctx.Status = storage.KubernetesStatus_Deleting
	}
	// cm.ctx.Store.UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

	if cm.conn == nil {
		conn, err := NewConnector(cm.ctx)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}
	cm.namer = namer{ctx: cm.ctx}

	exists, err := cm.findVPC()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if !exists {
		return errors.Newf("VPC %v not found for Cluster %v", cm.ctx.VpcId, cm.ctx.Name).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.ctx.StatusCause != "" {
		errs = append(errs, cm.ctx.StatusCause)
	}

	for _, ng := range cm.ctx.NodeGroups {
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
	for _, ng := range cm.ctx.NodeGroups {
		if err = cm.deleteLaunchConfiguration(cm.namer.AutoScalingGroupName(ng.Sku)); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if err = cm.deleteVolume(); err != nil {
		errs = append(errs, err.Error())
	}

	if req.ReleaseReservedIp && cm.ctx.MasterReservedIP != "" {
		if err = cm.releaseReservedIP(cm.ctx.MasterReservedIP); err != nil {
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

	if err = lib.DeleteARecords(cm.ctx); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.ctx.Status == storage.KubernetesStatus_Deleting {
			cm.ctx.StatusCause = strings.Join(errs, "\n")
		}
		errorhandlers.SendMailWithContextAndIgnore(cm.ctx, fmt.Errorf(strings.Join(errs, "\n")))
	}

	cm.ctx.Logger.Infof("Cluster %v deleted successfully", cm.ctx.Name)
	return nil
}

func (cluster *clusterManager) findVPC() (bool, error) {
	r1, err := cluster.conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
		VpcIds: []*string{
			types.StringP(cluster.ctx.VpcId),
		},
	})
	if err != nil {
		return false, errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	return len(r1.Vpcs) > 0, nil
}

func (cluster *clusterManager) deleteAutoScalingGroup(name string) error {
	_, err := cluster.conn.autoscale.DeleteAutoScalingGroup(&autoscaling.DeleteAutoScalingGroupInput{
		ForceDelete:          types.TrueP(),
		AutoScalingGroupName: types.StringP(name),
	})
	cluster.ctx.Logger.Infof("Auto scaling group %v is deleted for cluster %v", name, cluster.ctx.Name)
	return err
}

func (cluster *clusterManager) deleteLaunchConfiguration(name string) error {
	_, err := cluster.conn.autoscale.DeleteLaunchConfiguration(&autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: types.StringP(name),
	})
	cluster.ctx.Logger.Infof("Launch configuration %v os de;eted for cluster %v", name, cluster.ctx.Name)
	return err
}

func (cluster *clusterManager) deleteMaster() error {
	r1, err := cluster.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:Role"),
				Values: []*string{
					types.StringP(system.RoleKubernetesMaster),
				},
			},
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cluster.ctx.Name),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}

	masterInstances := make([]*string, 0)
	for _, reservation := range r1.Reservations {
		for _, instance := range reservation.Instances {
			masterInstances = append(masterInstances, instance.InstanceId)
		}
	}
	fmt.Printf("TerminateInstances %v", stringutil.Join(masterInstances, ","))
	cluster.ctx.Logger.Infof("Terminating master instance for cluster %v", cluster.ctx.Name)
	_, err = cluster.conn.ec2.TerminateInstances(&_ec2.TerminateInstancesInput{
		InstanceIds: masterInstances,
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	instanceInput := &_ec2.DescribeInstancesInput{
		InstanceIds: masterInstances,
	}
	err = cluster.conn.ec2.WaitUntilInstanceTerminated(instanceInput)
	fmt.Println(err, "--------------------<<<<<<<")
	cluster.ctx.Logger.Infof("Master instance for cluster %v is terminated", cluster.ctx.Name)
	return nil
}

func (cluster *clusterManager) ensureInstancesDeleted() error {
	const desiredState = "terminated"

	r1, err := cluster.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cluster.ctx.Name),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
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

		r2, err := cluster.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{InstanceIds: ris})
		if err != nil {
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
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

func (cluster *clusterManager) deleteSecurityGroup() error {
	r, err := cluster.conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cluster.ctx.VpcId),
				},
			},
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cluster.ctx.Name),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}

	for _, sg := range r.SecurityGroups {
		if len(sg.IpPermissions) > 0 {
			_, err := cluster.conn.ec2.RevokeSecurityGroupIngress(&_ec2.RevokeSecurityGroupIngressInput{
				GroupId:       sg.GroupId,
				IpPermissions: sg.IpPermissions,
			})
			if err != nil {
				return errors.FromErr(err).WithContext(cluster.ctx).Err()
			}
		}

		if len(sg.IpPermissionsEgress) > 0 {
			_, err := cluster.conn.ec2.RevokeSecurityGroupEgress(&_ec2.RevokeSecurityGroupEgressInput{
				GroupId:       sg.GroupId,
				IpPermissions: sg.IpPermissionsEgress,
			})
			if err != nil {
				return errors.FromErr(err).WithContext(cluster.ctx).Err()
			}
		}
	}

	for _, sg := range r.SecurityGroups {
		_, err := cluster.conn.ec2.DeleteSecurityGroup(&_ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
		}
	}
	cluster.ctx.Logger.Infof("Security groups for cluster %v is deleted", cluster.ctx.Name)
	return nil
}

func (cluster *clusterManager) deleteSubnetId() error {
	r, err := cluster.conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cluster.ctx.VpcId),
				},
			},
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cluster.ctx.Name),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	for _, subnet := range r.Subnets {
		_, err := cluster.conn.ec2.DeleteSubnet(&_ec2.DeleteSubnetInput{
			SubnetId: subnet.SubnetId,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
		}
		cluster.ctx.Logger.Infof("Subnet ID in VPC %v is deleted", *subnet.SubnetId)
	}
	return nil
}

func (cluster *clusterManager) deleteInternetGateway() error {
	r1, err := cluster.conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("attachment.vpc-id"),
				Values: []*string{
					types.StringP(cluster.ctx.VpcId),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	for _, igw := range r1.InternetGateways {
		_, err := cluster.conn.ec2.DetachInternetGateway(&_ec2.DetachInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
			VpcId:             types.StringP(cluster.ctx.VpcId),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
		}

		_, err = cluster.conn.ec2.DeleteInternetGateway(&_ec2.DeleteInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
		}
	}
	cluster.ctx.Logger.Infof("Internet gateway for cluster %v are deleted", cluster.ctx.Name)
	return nil
}

func (cluster *clusterManager) deleteRouteTable() error {
	r1, err := cluster.conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cluster.ctx.VpcId),
				},
			},
		},
	})

	if err != nil {
		fmt.Println(err)
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	for _, rt := range r1.RouteTables {
		mainTable := false
		for _, assoc := range rt.Associations {
			if _aws.BoolValue(assoc.Main) {
				mainTable = true
			} else {
				_, err := cluster.conn.ec2.DisassociateRouteTable(&_ec2.DisassociateRouteTableInput{
					AssociationId: assoc.RouteTableAssociationId,
				})
				if err != nil {
					fmt.Println(err)
					return errors.FromErr(err).WithContext(cluster.ctx).Err()
				}
			}
		}
		if !mainTable {
			_, err := cluster.conn.ec2.DeleteRouteTable(&_ec2.DeleteRouteTableInput{
				RouteTableId: rt.RouteTableId,
			})
			if err != nil {
				fmt.Println(err)
				return errors.FromErr(err).WithContext(cluster.ctx).Err()
			}
		}
	}
	cluster.ctx.Logger.Infof("Route tables for cluster %v are deleted", cluster.ctx.Name)
	return nil
}

func (cluster *clusterManager) deleteDHCPOption() error {
	_, err := cluster.conn.ec2.AssociateDhcpOptions(&_ec2.AssociateDhcpOptionsInput{
		VpcId:         types.StringP(cluster.ctx.VpcId),
		DhcpOptionsId: types.StringP("default"),
	})
	if err != nil {
		fmt.Println(err)
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}

	r, err := cluster.conn.ec2.DescribeDhcpOptions(&_ec2.DescribeDhcpOptionsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cluster.ctx.Name),
				},
			},
		},
	})
	for _, dhcp := range r.DhcpOptions {
		_, err = cluster.conn.ec2.DeleteDhcpOptions(&_ec2.DeleteDhcpOptionsInput{
			DhcpOptionsId: dhcp.DhcpOptionsId,
		})
		if err != nil {
			fmt.Println(err)
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
		}
	}
	cluster.ctx.Logger.Infof("DHCP options for cluster %v are deleted", cluster.ctx.Name)
	return err
}

func (cluster *clusterManager) deleteVpc() error {
	_, err := cluster.conn.ec2.DeleteVpc(&_ec2.DeleteVpcInput{
		VpcId: types.StringP(cluster.ctx.VpcId),
	})

	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	cluster.ctx.Logger.Infof("VPC for cluster %v is deleted", cluster.ctx.Name)
	return nil
}

func (cluster *clusterManager) deleteVolume() error {
	_, err := cluster.conn.ec2.DeleteVolume(&_ec2.DeleteVolumeInput{
		VolumeId: types.StringP(cluster.ctx.MasterDiskId),
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	cluster.ctx.Logger.Infof("Master instance volume for cluster %v is deleted", cluster.ctx.MasterDiskId)
	return nil
}

func (cluster *clusterManager) deleteSSHKey() error {
	var err error
	_, err = cluster.conn.ec2.DeleteKeyPair(&_ec2.DeleteKeyPairInput{
		KeyName: types.StringP(cluster.ctx.SSHKeyExternalID),
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	cluster.ctx.Logger.Infof("SSH key for cluster %v is deleted", cluster.ctx.MasterDiskId)
	//updates := &storage.SSHKey{IsDeleted: 1}
	//cond := &storage.SSHKey{PHID: cluster.ctx.SSHKeyPHID}
	// _, err = cluster.ctx.Store.Engine.Update(updates, cond)

	return err
}

func (cluster *clusterManager) releaseReservedIP(publicIP string) error {
	r1, err := cluster.conn.ec2.DescribeAddresses(&_ec2.DescribeAddressesInput{
		PublicIps: []*string{
			types.StringP(publicIP),
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}

	_, err = cluster.conn.ec2.ReleaseAddress(&_ec2.ReleaseAddressInput{
		AllocationId: r1.Addresses[0].AllocationId,
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	cluster.ctx.Logger.Infof("Elastic IP for cluster %v is deleted", cluster.ctx.Name)
	return nil
}

func (cluster *clusterManager) deleteNetworkInterface(vpcId string) error {
	r, err := cluster.conn.ec2.DescribeNetworkInterfaces(&_ec2.DescribeNetworkInterfacesInput{
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
		return errors.FromErr(err).WithContext(cluster.ctx).Err()
	}
	for _, iface := range r.NetworkInterfaces {
		_, err = cluster.conn.ec2.DetachNetworkInterface(&_ec2.DetachNetworkInterfaceInput{
			AttachmentId: iface.Attachment.AttachmentId,
			Force:        types.TrueP(),
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
		}

		time.Sleep(1 * time.Minute)
		_, err = cluster.conn.ec2.DeleteNetworkInterface(&_ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: iface.NetworkInterfaceId,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cluster.ctx).Err()
		}
	}
	cluster.ctx.Logger.Infof("Network interfaces for cluster %v are deleted", cluster.ctx.Name)
	return nil
}

func (cluster *clusterManager) deleteBucket() error {
	// http://docs.aws.amazon.com/AmazonS3/latest/dev/delete-or-empty-bucket.html#delete-bucket-awscli
	cluster.ctx.Logger.Infof("Deleting startupconfig bucket for cluster %v", cluster.ctx.Name)
	var timeout int64 = 30 * 60 // Give max 30 min to empty the bucket
	start := time.Now().Unix()

	for {
		r1, err := cluster.conn.s3.ListObjectsV2(&_s3.ListObjectsV2Input{
			Bucket: types.StringP(cluster.ctx.BucketName),
		})
		if err == nil && len(r1.Contents) > 0 {
			oIds := make([]*_s3.ObjectIdentifier, len(r1.Contents))
			for i, obj := range r1.Contents {
				oIds[i] = &_s3.ObjectIdentifier{Key: obj.Key}
			}
			cluster.conn.s3.DeleteObjects(&_s3.DeleteObjectsInput{
				Bucket: types.StringP(cluster.ctx.BucketName),
				Delete: &_s3.Delete{Objects: oIds},
			})
		}
		if len(r1.Contents) == 0 || (time.Now().Unix() > start+timeout) {
			break
		}
	}

	for {
		r1, err := cluster.conn.s3.ListObjectVersions(&_s3.ListObjectVersionsInput{
			Bucket: types.StringP(cluster.ctx.BucketName),
		})
		if err == nil && len(r1.DeleteMarkers) > 0 {
			oIds := make([]*_s3.ObjectIdentifier, len(r1.DeleteMarkers))
			for i, obj := range r1.DeleteMarkers {
				oIds[i] = &_s3.ObjectIdentifier{Key: obj.Key, VersionId: obj.VersionId}
			}
			cluster.conn.s3.DeleteObjects(&_s3.DeleteObjectsInput{
				Bucket: types.StringP(cluster.ctx.BucketName),
				Delete: &_s3.Delete{Objects: oIds},
			})
		}
		if len(r1.DeleteMarkers) == 0 || (time.Now().Unix() > start+timeout) {
			break
		}
	}

	_, err := cluster.conn.s3.DeleteBucket(&_s3.DeleteBucketInput{
		Bucket: types.StringP(cluster.ctx.BucketName),
	})
	return err
}
