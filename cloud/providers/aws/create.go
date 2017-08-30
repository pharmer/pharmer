package aws

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"
	// "github.com/appscode/pharmer/templates"
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
	// "github.com/appscode/pharmer/templates"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_iam "github.com/aws/aws-sdk-go/service/iam"
)

const (
	preTagDelay = 5 * time.Second
)

func (cm *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn, err = NewConnector(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)

	defer func(releaseReservedIp bool) {
		if cm.cluster.Status.Phase == api.ClusterPhasePending {
			cm.cluster.Status.Phase = api.ClusterPhaseFailing
		}
		cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		cloud.Store(cm.ctx).Instances(cm.cluster.Name).SaveInstances(cm.ins.Instances)
		cloud.Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterPhaseReady {
			cloud.Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.Delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")

	if err = cm.conn.detectUbuntuImage(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Spec.InstanceImage = cm.conn.cluster.Spec.InstanceImage
	// TODO: FixIt!
	//cm.cluster.Spec.RootDeviceName = cm.conn.cluster.Spec.RootDeviceName
	//fmt.Println(cm.cluster.Spec.InstanceImage, cm.cluster.Spec.RootDeviceName, "---------------*********")

	if err = cm.ensureIAMProfile(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.importPublicKey(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupVpc(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.createDHCPOptionSet(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupSubnet(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupInternetGateway(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupRouteTable(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupSecurityGroups(); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterInstance, err := cm.startMaster()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	for _, ng := range req.NodeGroups {
		igm := &InstanceGroupManager{
			cm: cm,
			instance: cloud.Instance{
				Type: cloud.InstanceType{
					ContextVersion: cm.cluster.Spec.ResourceVersion,
					Sku:            ng.Sku,

					Master:       false,
					SpotInstance: false,
				},
				Stats: cloud.GroupStats{
					Count: ng.Count,
				},
			},
		}
		igm.AdjustInstanceGroup()
	}

	cloud.Logger(cm.ctx).Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := cloud.EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// wait for nodes to start
	if err := cloud.ProbeKubeAPI(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = cloud.CheckComponentStatuses(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = cloud.WaitForReadyNodes(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------------------------------------------------
	cloud.Logger(cm.ctx).Info("Listing autoscaling groups")
	groups := make([]*string, 0)
	for _, ng := range req.NodeGroups {
		groups = append(groups, types.StringP(cm.namer.AutoScalingGroupName(ng.Sku)))
	}
	r2, err := cm.conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groups,
	})
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	fmt.Println(r2)
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	for _, group := range r2.AutoScalingGroups {
		for _, instance := range group.Instances {
			ki, err := cm.newKubeInstance(*instance.InstanceId)
			ki.Spec.Role = api.RoleKubernetesPool
			cm.ins.Instances = append(cm.ins.Instances, ki)
			if err != nil {
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
		}
	}

	// detect-master
	// wait-master: via curl call polling
	// build-config

	//  # KUBE_SHARE_MASTER is used to add nodes to an existing master
	//  if [[ "${KUBE_SHARE_MASTER:-}" == "true" ]]; then
	//    detect-master
	//    start-nodes
	//    wait-nodes
	//  else
	//    start-master
	//    start-nodes
	//    wait-nodes
	//    wait-master
	//
	//    # Build ~/.kube/config
	//    build-config
	//  fi
	// check-cluster
	cm.cluster.Status.Phase = api.ClusterPhaseReady
	return nil
}

func (cm *ClusterManager) ensureIAMProfile() error {
	r1, _ := cm.conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &cm.cluster.Spec.IAMProfileMaster})
	if r1.InstanceProfile == nil {
		err := cm.createIAMProfile(cm.cluster.Spec.IAMProfileMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Master instance profile %v created", cm.cluster.Spec.IAMProfileMaster)
	}
	r2, _ := cm.conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &cm.cluster.Spec.IAMProfileNode})
	if r2.InstanceProfile == nil {
		err := cm.createIAMProfile(cm.cluster.Spec.IAMProfileNode)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Node instance profile %v created", cm.cluster.Spec.IAMProfileNode)
	}
	return nil
}

func (cm *ClusterManager) createIAMProfile(key string) error {
	//rootDir := "kubernetes/aws/iam/"
	role := "" // TODO(tamal); FixIt!  templates.AssetText(rootDir + key + "-role.json")
	r1, err := cm.conn.iam.CreateRole(&_iam.CreateRoleInput{
		RoleName:                 &key,
		AssumeRolePolicyDocument: &role,
	})
	cloud.Logger(cm.ctx).Debug("Created IAM role", r1, err)
	cloud.Logger(cm.ctx).Infof("IAM role %v created", key)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	policy := "" // TODO(tamal); FixIt!  templates.AssetText(rootDir + key + "-policy.json")
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	r2, err := cm.conn.iam.PutRolePolicy(&_iam.PutRolePolicyInput{
		RoleName:       &key,
		PolicyName:     &key,
		PolicyDocument: &policy,
	})
	cloud.Logger(cm.ctx).Debug("Created IAM role-policy", r2, err)
	cloud.Logger(cm.ctx).Infof("IAM role-policy %v created", key)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	r3, err := cm.conn.iam.CreateInstanceProfile(&_iam.CreateInstanceProfileInput{
		InstanceProfileName: &key,
	})
	cloud.Logger(cm.ctx).Debug("Created IAM instance-policy", r3, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("IAM instance-policy %v created", key)

	r4, err := cm.conn.iam.AddRoleToInstanceProfile(&_iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: &key,
		RoleName:            &key,
	})
	cloud.Logger(cm.ctx).Debug("Added IAM role to instance-policy", r4, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("IAM role %v added to instance-policy %v", key, key)
	return nil
}

func (cm *ClusterManager) importPublicKey() error {
	resp, err := cm.conn.ec2.ImportKeyPair(&_ec2.ImportKeyPairInput{
		KeyName:           types.StringP(cm.cluster.Spec.SSHKeyExternalID),
		PublicKeyMaterial: cm.cluster.Spec.SSHKey.PublicKey,
	})
	cloud.Logger(cm.ctx).Debug("Imported SSH key", resp, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// TODO ignore "InvalidKeyPair.Duplicate" error
	if err != nil {
		cloud.Logger(cm.ctx).Info("Error importing public key", resp, err)
		//os.Exit(1)
		return errors.FromErr(err).WithContext(cm.ctx).Err()

	}
	cloud.Logger(cm.ctx).Infof("SSH key with (AWS) fingerprint %v imported", cm.cluster.Spec.SSHKey.AwsFingerprint)

	return nil
}

func (cm *ClusterManager) setupVpc() error {
	cloud.Logger(cm.ctx).Infof("Checking VPC tagged with %v", cm.cluster.Name)
	r1, err := cm.conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:Name"),
				Values: []*string{
					types.StringP(cm.namer.VPCName()),
				},
			},
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.cluster.Name), // Tag by Name or PHID?
				},
			},
		},
	})
	cloud.Logger(cm.ctx).Debug("VPC described", r1, err)
	if len(r1.Vpcs) > 1 {
		cm.cluster.Spec.VpcId = *r1.Vpcs[0].VpcId
		cloud.Logger(cm.ctx).Infof("VPC %v found", cm.cluster.Spec.VpcId)
	}

	cloud.Logger(cm.ctx).Info("No VPC found, creating new VPC")
	r2, err := cm.conn.ec2.CreateVpc(&_ec2.CreateVpcInput{
		CidrBlock: types.StringP(cm.cluster.Spec.VpcCidr),
	})
	cloud.Logger(cm.ctx).Debug("VPC created", r2, err)
	//errorutil.EOE(err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("VPC %v created", *r2.Vpc.VpcId)
	cm.cluster.Spec.VpcId = *r2.Vpc.VpcId

	r3, err := cm.conn.ec2.ModifyVpcAttribute(&_ec2.ModifyVpcAttributeInput{
		VpcId: types.StringP(cm.cluster.Spec.VpcId),
		EnableDnsSupport: &_ec2.AttributeBooleanValue{
			Value: types.TrueP(),
		},
	})
	cloud.Logger(cm.ctx).Debug("DNS support enabled", r3, err)
	cloud.Logger(cm.ctx).Infof("Enabled DNS support for VPCID %v", cm.cluster.Spec.VpcId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	r4, err := cm.conn.ec2.ModifyVpcAttribute(&_ec2.ModifyVpcAttributeInput{
		VpcId: types.StringP(cm.cluster.Spec.VpcId),
		EnableDnsHostnames: &_ec2.AttributeBooleanValue{
			Value: types.TrueP(),
		},
	})
	cloud.Logger(cm.ctx).Debug("DNS hostnames enabled", r4, err)
	cloud.Logger(cm.ctx).Infof("Enabled DNS hostnames for VPCID %v", cm.cluster.Spec.VpcId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	time.Sleep(preTagDelay)
	cm.addTag(cm.cluster.Spec.VpcId, "Name", cm.namer.VPCName())
	cm.addTag(cm.cluster.Spec.VpcId, "KubernetesCluster", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) addTag(id string, key string, value string) error {
	resp, err := cm.conn.ec2.CreateTags(&_ec2.CreateTagsInput{
		Resources: []*string{
			types.StringP(id),
		},
		Tags: []*_ec2.Tag{
			{
				Key:   types.StringP(key),
				Value: types.StringP(value),
			},
		},
	})
	cloud.Logger(cm.ctx).Debug("Added tag ", resp, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Added tag %v:%v to id %v", key, value, id)
	return nil
}

func (cm *ClusterManager) createDHCPOptionSet() error {
	optionSetDomain := fmt.Sprintf("%v.compute.internal", cm.cluster.Spec.Region)
	if cm.cluster.Spec.Region == "us-east-1" {
		optionSetDomain = "ec2.internal"
	}
	r1, err := cm.conn.ec2.CreateDhcpOptions(&_ec2.CreateDhcpOptionsInput{
		DhcpConfigurations: []*_ec2.NewDhcpConfiguration{
			{
				Key:    types.StringP("domain-name"),
				Values: []*string{types.StringP(optionSetDomain)},
			},
			{
				Key:    types.StringP("domain-name-servers"),
				Values: []*string{types.StringP("AmazonProvidedDNS")},
			},
		},
	})
	cloud.Logger(cm.ctx).Debug("Created DHCP options ", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("DHCP options created with id %v", *r1.DhcpOptions.DhcpOptionsId)
	cm.cluster.Spec.DHCPOptionsId = *r1.DhcpOptions.DhcpOptionsId

	time.Sleep(preTagDelay)
	cm.addTag(cm.cluster.Spec.DHCPOptionsId, "Name", cm.namer.DHCPOptionsName())
	cm.addTag(cm.cluster.Spec.DHCPOptionsId, "KubernetesCluster", cm.cluster.Name)

	r2, err := cm.conn.ec2.AssociateDhcpOptions(&_ec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: types.StringP(cm.cluster.Spec.DHCPOptionsId),
		VpcId:         types.StringP(cm.cluster.Spec.VpcId),
	})
	cloud.Logger(cm.ctx).Debug("Associated DHCP options ", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("DHCP options %v associated with %v", cm.cluster.Spec.DHCPOptionsId, cm.cluster.Spec.VpcId)

	return nil
}

func (cm *ClusterManager) setupSubnet() error {
	cloud.Logger(cm.ctx).Info("Checking for existing subnet")
	r1, err := cm.conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.cluster.Name),
				},
			},
			{
				Name: types.StringP("availabilityZone"),
				Values: []*string{
					types.StringP(cm.cluster.Spec.Zone),
				},
			},
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cm.cluster.Spec.VpcId),
				},
			},
		},
	})
	cloud.Logger(cm.ctx).Debug("Retrieved subnet", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if len(r1.Subnets) == 0 {
		cloud.Logger(cm.ctx).Info("No subnet found, creating new subnet")
		r2, err := cm.conn.ec2.CreateSubnet(&_ec2.CreateSubnetInput{
			CidrBlock:        types.StringP(cm.cluster.Spec.SubnetCidr),
			VpcId:            types.StringP(cm.cluster.Spec.VpcId),
			AvailabilityZone: types.StringP(cm.cluster.Spec.Zone),
		})
		cloud.Logger(cm.ctx).Debug("Created subnet", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Subnet %v created", *r2.Subnet.SubnetId)
		cm.cluster.Spec.SubnetId = *r2.Subnet.SubnetId

		time.Sleep(preTagDelay)
		cm.addTag(cm.cluster.Spec.SubnetId, "KubernetesCluster", cm.cluster.Name)

	} else {
		cm.cluster.Spec.SubnetId = *r1.Subnets[0].SubnetId
		existingCIDR := *r1.Subnets[0].CidrBlock
		cloud.Logger(cm.ctx).Infof("Subnet %v found with CIDR %v", cm.cluster.Spec.SubnetId, existingCIDR)

		cloud.Logger(cm.ctx).Infof("Retrieving VPC %v", cm.cluster.Spec.VpcId)
		r3, err := cm.conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
			VpcIds: []*string{types.StringP(cm.cluster.Spec.VpcId)},
		})
		cloud.Logger(cm.ctx).Debug("Retrieved VPC", r3, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		octets := strings.Split(*r3.Vpcs[0].CidrBlock, ".")
		cm.cluster.Spec.VpcCidrBase = octets[0] + "." + octets[1]
		cm.cluster.Spec.MasterInternalIP = cm.cluster.Spec.VpcCidrBase + ".0" + cm.cluster.Spec.MasterIPSuffix
		cloud.Logger(cm.ctx).Infof("Assuming MASTER_INTERNAL_IP=%v", cm.cluster.Spec.MasterInternalIP)
	}
	return nil
}

func (cm *ClusterManager) setupInternetGateway() error {
	cloud.Logger(cm.ctx).Infof("Checking IGW with attached VPCID %v", cm.cluster.Spec.VpcId)
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
	cloud.Logger(cm.ctx).Debug("Retrieved IGW", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if len(r1.InternetGateways) == 0 {
		cloud.Logger(cm.ctx).Info("No IGW found, creating new IGW")
		r2, err := cm.conn.ec2.CreateInternetGateway(&_ec2.CreateInternetGatewayInput{})
		cloud.Logger(cm.ctx).Debug("Created IGW", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.cluster.Spec.IGWId = *r2.InternetGateway.InternetGatewayId
		time.Sleep(preTagDelay)
		cloud.Logger(cm.ctx).Infof("IGW %v created", cm.cluster.Spec.IGWId)

		r3, err := cm.conn.ec2.AttachInternetGateway(&_ec2.AttachInternetGatewayInput{
			InternetGatewayId: types.StringP(cm.cluster.Spec.IGWId),
			VpcId:             types.StringP(cm.cluster.Spec.VpcId),
		})
		cloud.Logger(cm.ctx).Debug("Attached IGW to VPC", r3, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Attached IGW %v to VPCID %v", cm.cluster.Spec.IGWId, cm.cluster.Spec.VpcId)

		cm.addTag(cm.cluster.Spec.IGWId, "Name", cm.namer.InternetGatewayName())
		cm.addTag(cm.cluster.Spec.IGWId, "KubernetesCluster", cm.cluster.Name)
	} else {
		cm.cluster.Spec.IGWId = *r1.InternetGateways[0].InternetGatewayId
		cloud.Logger(cm.ctx).Infof("IGW %v found", cm.cluster.Spec.IGWId)
	}
	return nil
}

func (cm *ClusterManager) setupRouteTable() error {
	cloud.Logger(cm.ctx).Infof("Checking route table for VPCID %v", cm.cluster.Spec.VpcId)
	r1, err := cm.conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
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
	cloud.Logger(cm.ctx).Debug("Attached IGW to VPC", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if len(r1.RouteTables) == 0 {
		cloud.Logger(cm.ctx).Infof("No route table found for VPCID %v, creating new route table", cm.cluster.Spec.VpcId)
		r2, err := cm.conn.ec2.CreateRouteTable(&_ec2.CreateRouteTableInput{
			VpcId: types.StringP(cm.cluster.Spec.VpcId),
		})
		cloud.Logger(cm.ctx).Debug("Created route table", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		cm.cluster.Spec.RouteTableId = *r2.RouteTable.RouteTableId
		cloud.Logger(cm.ctx).Infof("Route table %v created", cm.cluster.Spec.RouteTableId)
		time.Sleep(preTagDelay)
		cm.addTag(cm.cluster.Spec.RouteTableId, "KubernetesCluster", cm.cluster.Name)

	} else {
		cm.cluster.Spec.RouteTableId = *r1.RouteTables[0].RouteTableId
		cloud.Logger(cm.ctx).Infof("Route table %v found", cm.cluster.Spec.RouteTableId)
	}

	r3, err := cm.conn.ec2.AssociateRouteTable(&_ec2.AssociateRouteTableInput{
		RouteTableId: types.StringP(cm.cluster.Spec.RouteTableId),
		SubnetId:     types.StringP(cm.cluster.Spec.SubnetId),
	})
	cloud.Logger(cm.ctx).Debug("Associating route table to subnet", r3, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Route table %v associated to subnet %v", cm.cluster.Spec.RouteTableId, cm.cluster.Spec.SubnetId)

	r4, err := cm.conn.ec2.CreateRoute(&_ec2.CreateRouteInput{
		RouteTableId:         types.StringP(cm.cluster.Spec.RouteTableId),
		DestinationCidrBlock: types.StringP("0.0.0.0/0"),
		GatewayId:            types.StringP(cm.cluster.Spec.IGWId),
	})
	cloud.Logger(cm.ctx).Debug("Added route to route table", r4, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Route added to route table %v", cm.cluster.Spec.RouteTableId)
	return nil
}

func (cm *ClusterManager) setupSecurityGroups() error {
	var ok bool
	var err error
	if cm.cluster.Spec.MasterSGId, ok, err = cm.getSecurityGroupId(cm.cluster.Spec.MasterSGName); !ok {
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.createSecurityGroup(cm.cluster.Spec.MasterSGName, "Kubernetes security group applied to master instance")
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Master security group %v created", cm.cluster.Spec.MasterSGName)
	}
	if cm.cluster.Spec.NodeSGId, ok, err = cm.getSecurityGroupId(cm.cluster.Spec.NodeSGName); !ok {
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.createSecurityGroup(cm.cluster.Spec.NodeSGName, "Kubernetes security group applied to node instances")
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Naster security group %v created", cm.cluster.Spec.NodeSGName)
	}

	err = cm.detectSecurityGroups()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cloud.Logger(cm.ctx).Info("Masters can talk to master")
	err = cm.autohrizeIngressBySGID(cm.cluster.Spec.MasterSGId, cm.cluster.Spec.MasterSGId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cloud.Logger(cm.ctx).Info("Nodes can talk to nodes")
	err = cm.autohrizeIngressBySGID(cm.cluster.Spec.NodeSGId, cm.cluster.Spec.NodeSGId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cloud.Logger(cm.ctx).Info("Masters and nodes can talk to each other")
	err = cm.autohrizeIngressBySGID(cm.cluster.Spec.MasterSGId, cm.cluster.Spec.NodeSGId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.autohrizeIngressBySGID(cm.cluster.Spec.NodeSGId, cm.cluster.Spec.MasterSGId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// TODO(justinsb): Would be fairly easy to replace 0.0.0.0/0 in these rules

	cloud.Logger(cm.ctx).Info("SSH is opened to the world")
	err = cm.autohrizeIngressByPort(cm.cluster.Spec.MasterSGId, 22)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.autohrizeIngressByPort(cm.cluster.Spec.NodeSGId, 22)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cloud.Logger(cm.ctx).Info("HTTPS to the master is allowed (for API access)")
	err = cm.autohrizeIngressByPort(cm.cluster.Spec.MasterSGId, 443)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.autohrizeIngressByPort(cm.cluster.Spec.MasterSGId, 6443)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *ClusterManager) getSecurityGroupId(groupName string) (string, bool, error) {
	cloud.Logger(cm.ctx).Infof("Checking security group %v", groupName)
	r1, err := cm.conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cm.cluster.Spec.VpcId),
				},
			},
			{
				Name: types.StringP("group-name"),
				Values: []*string{
					types.StringP(groupName),
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
	cloud.Logger(cm.ctx).Debug("Retrieved security group", r1, err)
	if err != nil {
		return "", false, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if len(r1.SecurityGroups) == 0 {
		cloud.Logger(cm.ctx).Infof("No security group %v found", groupName)
		return "", false, nil
	}
	cloud.Logger(cm.ctx).Infof("Security group %v found", groupName)
	return *r1.SecurityGroups[0].GroupId, true, nil
}

func (cm *ClusterManager) createSecurityGroup(groupName string, description string) error {
	cloud.Logger(cm.ctx).Infof("Creating security group %v", groupName)
	r2, err := cm.conn.ec2.CreateSecurityGroup(&_ec2.CreateSecurityGroupInput{
		GroupName:   types.StringP(groupName),
		Description: types.StringP(description),
		VpcId:       types.StringP(cm.cluster.Spec.VpcId),
	})
	cloud.Logger(cm.ctx).Debug("Created security group", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	time.Sleep(preTagDelay)
	err = cm.addTag(*r2.GroupId, "KubernetesCluster", cm.cluster.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *ClusterManager) detectSecurityGroups() error {
	var ok bool
	var err error
	if cm.cluster.Spec.MasterSGId == "" {
		if cm.cluster.Spec.MasterSGId, ok, err = cm.getSecurityGroupId(cm.cluster.Spec.MasterSGName); !ok {
			return errors.New("Could not detect Kubernetes master security group.  Make sure you've launched a cluster with appctl").WithContext(cm.ctx).Err()
		} else {
			cloud.Logger(cm.ctx).Infof("Master security group %v with id %v detected", cm.cluster.Spec.MasterSGName, cm.cluster.Spec.MasterSGId)
		}
	}
	if cm.cluster.Spec.NodeSGId == "" {
		if cm.cluster.Spec.NodeSGId, ok, err = cm.getSecurityGroupId(cm.cluster.Spec.NodeSGName); !ok {
			return errors.New("Could not detect Kubernetes node security group.  Make sure you've launched a cluster with appctl").WithContext(cm.ctx).Err()
		} else {
			cloud.Logger(cm.ctx).Infof("Node security group %v with id %v detected", cm.cluster.Spec.NodeSGName, cm.cluster.Spec.NodeSGId)
		}
	}
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *ClusterManager) autohrizeIngressBySGID(groupID string, srcGroup string) error {
	r1, err := cm.conn.ec2.AuthorizeSecurityGroupIngress(&_ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: types.StringP(groupID),
		IpPermissions: []*_ec2.IpPermission{
			{
				IpProtocol: types.StringP("-1"),
				UserIdGroupPairs: []*_ec2.UserIdGroupPair{
					{
						GroupId: types.StringP(srcGroup),
					},
				},
			},
		},
	})
	cloud.Logger(cm.ctx).Debug("Authorized ingress", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Ingress authorized into SG %v from SG %v", groupID, srcGroup)
	return nil
}

func (cm *ClusterManager) autohrizeIngressByPort(groupID string, port int64) error {
	r1, err := cm.conn.ec2.AuthorizeSecurityGroupIngress(&_ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: types.StringP(groupID),
		IpPermissions: []*_ec2.IpPermission{
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(port),
				IpRanges: []*_ec2.IpRange{
					{
						CidrIp: types.StringP("0.0.0.0/0"),
					},
				},
				ToPort: types.Int64P(port),
			},
		},
	})
	cloud.Logger(cm.ctx).Debug("Authorized ingress", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Authorized ingress into SG %v via port %v", groupID, port)
	return nil
}

//
// -------------------------------------
//
func (cm *ClusterManager) startMaster() (*api.Instance, error) {
	var err error
	// TODO: FixIt!
	//cm.cluster.Spec.MasterDiskId, err = cm.ensurePd(cm.namer.MasterPDName(), cm.cluster.Spec.MasterDiskType, cm.cluster.Spec.MasterDiskSize)
	//if err != nil {
	//	return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	//}
	err = cm.reserveIP()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx, _ = cloud.GenClusterCerts(cm.ctx, cm.cluster)
	cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster) // needed for master start-up config

	masterInstanceID, err := cm.createMasterInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleKubernetesMaster)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Info("Waiting for master instance to be ready")
	// We are not able to add an elastic ip, a route or volume to the instance until that instance is in "running" state.
	err = cm.waitForInstanceState(masterInstanceID, "running")
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Info("Master instance is ready")
	if cm.cluster.Spec.MasterReservedIP != "" {
		err = cm.assignIPToInstance(masterInstanceID)
		if err != nil {
			return nil, errors.FromErr(err).WithMessage("failed to assign ip").WithContext(cm.ctx).Err()
		}
	}

	// TODO check setting master IP is set properly
	masterInstance, err := cm.newKubeInstance(masterInstanceID) // sets external IP
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Spec.Role = api.RoleKubernetesMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.ExternalIP
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		return masterInstance, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	_, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster) // needed for node start-up config to get master_internal_ip
	// This is a race between instance start and volume attachment.
	// There appears to be no way to start an AWS instance with a volume attached.
	// To work around this, we wait for volume to be ready in setup-master-pd.sh
	if err != nil {
		return masterInstance, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	/*
		r1, err := cm.conn.ec2.AttachVolume(&_ec2.AttachVolumeInput{
			VolumeId:   types.StringP(cm.cluster.Spec.MasterDiskId),
			Device:     types.StringP("/dev/sdb"),
			InstanceId: types.StringP(masterInstanceID),
		})
		cloud.Logger(cm.ctx).Debug("Attached persistent data volume to master", r1, err)
		if err != nil {
			return masterInstance, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cloud.Logger(cm.ctx).Infof("Persistent data volume %v attatched to master", cm.cluster.Spec.MasterDiskId)
	*/

	time.Sleep(15 * time.Second)
	r2, err := cm.conn.ec2.CreateRoute(&_ec2.CreateRouteInput{
		RouteTableId:         types.StringP(cm.cluster.Spec.RouteTableId),
		DestinationCidrBlock: types.StringP(cm.cluster.Spec.MasterIPRange),
		InstanceId:           types.StringP(masterInstanceID),
	})
	cloud.Logger(cm.ctx).Debug("Created route to master", r2, err)
	if err != nil {
		return masterInstance, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Master route to route table %v for ip %v created", cm.cluster.Spec.RouteTableId, masterInstanceID)
	return masterInstance, nil
}

func (cm *ClusterManager) ensurePd(name, diskType string, sizeGb int64) (string, error) {
	volumeId, err := cm.findPD(name)
	if err != nil {
		return volumeId, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if volumeId == "" {
		// name := cluster.Spec.ctx.KubernetesMasterName + "-pd"
		r1, err := cm.conn.ec2.CreateVolume(&_ec2.CreateVolumeInput{
			AvailabilityZone: &cm.cluster.Spec.Zone,
			VolumeType:       &diskType,
			Size:             types.Int64P(sizeGb),
		})
		cloud.Logger(cm.ctx).Debug("Created master pd", r1, err)
		if err != nil {
			return "", errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		volumeId = *r1.VolumeId
		cloud.Logger(cm.ctx).Infof("Master disk with size %vGB, type %v created", cm.cluster.Spec.MasterDiskSize, cm.cluster.Spec.MasterDiskType)

		time.Sleep(preTagDelay)
		err = cm.addTag(volumeId, "Name", name)
		if err != nil {
			return volumeId, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.addTag(volumeId, "KubernetesCluster", cm.cluster.Name)
		if err != nil {
			return volumeId, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	return volumeId, nil
}

func (cm *ClusterManager) findPD(name string) (string, error) {
	// name := cluster.Spec.ctx.KubernetesMasterName + "-pd"
	cloud.Logger(cm.ctx).Infof("Searching master pd %v", name)
	r1, err := cm.conn.ec2.DescribeVolumes(&_ec2.DescribeVolumesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("availability-zone"),
				Values: []*string{
					types.StringP(cm.cluster.Spec.Zone),
				},
			},
			{
				Name: types.StringP("tag:Name"),
				Values: []*string{
					types.StringP(name),
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
	cloud.Logger(cm.ctx).Debug("Retrieved master pd", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if len(r1.Volumes) > 0 {
		cloud.Logger(cm.ctx).Infof("Found master pd %v", name)
		return *r1.Volumes[0].VolumeId, nil
	}
	cloud.Logger(cm.ctx).Infof("Master pd %v not found", name)
	return "", nil
}

func (cm *ClusterManager) reserveIP() error {
	// Check that MASTER_RESERVED_IP looks like an IPv4 address
	// if match, _ := regexp.MatchString("^[0-9]+.[0-9]+.[0-9]+.[0-9]+$", cluster.Spec.ctx.MasterReservedIP); !match {
	if cm.cluster.Spec.MasterReservedIP == "auto" {
		r1, err := cm.conn.ec2.AllocateAddress(&_ec2.AllocateAddressInput{
			Domain: types.StringP("vpc"),
		})
		cloud.Logger(cm.ctx).Debug("Allocated elastic IP", r1, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		time.Sleep(5 * time.Second)
		cm.cluster.Spec.MasterReservedIP = *r1.PublicIp
		cloud.Logger(cm.ctx).Infof("Elastic IP %v allocated", cm.cluster.Spec.MasterReservedIP)
	}
	return nil
}

func (cm *ClusterManager) createMasterInstance(instanceName string, role string) (string, error) {
	kubeStarter, err := cloud.RenderStartupScript(cm.ctx, cm.cluster, api.RoleKubernetesMaster)
	if err != nil {
		return "", err
	}
	req := &_ec2.RunInstancesInput{
		ImageId:  types.StringP(cm.cluster.Spec.InstanceImage),
		MaxCount: types.Int64P(1),
		MinCount: types.Int64P(1),
		//// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html
		//BlockDeviceMappings: []*_ec2.BlockDeviceMapping{
		//	// MASTER_BLOCK_DEVICE_MAPPINGS
		//	{
		//		// https://github.com/appscode/kubernetes/blob/55d9dec8eb5eb02e1301045b7b81bbac689c86a1/cluster/aws/util.sh#L397
		//		DeviceName: types.StringP(cm.cluster.Spec.RootDeviceName),
		//		Ebs: &_ec2.EbsBlockDevice{
		//			DeleteOnTermination: types.TrueP(),
		//			VolumeSize:          types.Int64P(cm.cluster.Spec.MasterDiskSize),
		//			VolumeType:          types.StringP(cm.cluster.Spec.MasterDiskType),
		//		},
		//	},
		//	// EPHEMERAL_BLOCK_DEVICE_MAPPINGS
		//	{
		//		DeviceName:  types.StringP("/dev/sdc"),
		//		VirtualName: types.StringP("ephemeral0"),
		//	},
		//	{
		//		DeviceName:  types.StringP("/dev/sdd"),
		//		VirtualName: types.StringP("ephemeral1"),
		//	},
		//	{
		//		DeviceName:  types.StringP("/dev/sde"),
		//		VirtualName: types.StringP("ephemeral2"),
		//	},
		//	{
		//		DeviceName:  types.StringP("/dev/sdf"),
		//		VirtualName: types.StringP("ephemeral3"),
		//	},
		//},
		IamInstanceProfile: &_ec2.IamInstanceProfileSpecification{
			Name: types.StringP(cm.cluster.Spec.IAMProfileMaster),
		},
		InstanceType: types.StringP(cm.cluster.Spec.MasterSKU),
		KeyName:      types.StringP(cm.cluster.Spec.SSHKeyExternalID),
		Monitoring: &_ec2.RunInstancesMonitoringEnabled{
			Enabled: types.TrueP(),
		},
		NetworkInterfaces: []*_ec2.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: types.TrueP(),
				DeleteOnTermination:      types.TrueP(),
				DeviceIndex:              types.Int64P(0),
				Groups: []*string{
					types.StringP(cm.cluster.Spec.MasterSGId),
				},
				PrivateIpAddresses: []*_ec2.PrivateIpAddressSpecification{
					{
						PrivateIpAddress: types.StringP(cm.cluster.Spec.MasterInternalIP),
						Primary:          types.TrueP(),
					},
				},
				SubnetId: types.StringP(cm.cluster.Spec.SubnetId),
			},
		},
		UserData: types.StringP(base64.StdEncoding.EncodeToString([]byte(kubeStarter))),
	}
	r1, err := cm.conn.ec2.RunInstances(req)
	cloud.Logger(cm.ctx).Debug("Created instance", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Instance %v created with role %v", instanceName, role)
	instanceID := *r1.Instances[0].InstanceId
	time.Sleep(preTagDelay)

	err = cm.addTag(instanceID, "Name", cm.cluster.Spec.KubernetesMasterName)
	if err != nil {
		return instanceID, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.addTag(instanceID, "Role", role)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.addTag(instanceID, "KubernetesCluster", cm.cluster.Name)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return instanceID, nil
}

func (cm *ClusterManager) getInstancePublicIP(instanceID string) (string, bool, error) {
	r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		InstanceIds: []*string{types.StringP(instanceID)},
	})
	cloud.Logger(cm.ctx).Debug("Retrieved Public IP for Instance", r1, err)
	if err != nil {
		return "", false, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if r1.Reservations != nil && r1.Reservations[0].Instances != nil && r1.Reservations[0].Instances[0].NetworkInterfaces != nil {
		cloud.Logger(cm.ctx).Infof("Public ip for instance id %v retrieved", instanceID)
		return *r1.Reservations[0].Instances[0].NetworkInterfaces[0].Association.PublicIp, true, nil
	}
	return "", false, nil
}

func (cm *ClusterManager) listInstances(groupName string) ([]*api.Instance, error) {
	r2, err := cm.conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			types.StringP(groupName),
		},
	})
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances := make([]*api.Instance, 0)
	for _, group := range r2.AutoScalingGroups {
		for _, instance := range group.Instances {
			ki, err := cm.newKubeInstance(*instance.InstanceId)
			if err != nil {
				return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			ki.Spec.Role = api.RoleKubernetesPool
			instances = append(instances, ki)
		}
	}
	return instances, nil
}
func (cm *ClusterManager) newKubeInstance(instanceID string) (*api.Instance, error) {
	r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		InstanceIds: []*string{types.StringP(instanceID)},
	})
	cloud.Logger(cm.ctx).Debug("Retrieved instance ", r1, err)
	if err != nil {
		return nil, cloud.InstanceNotFound
	}

	// Don't reassign internal_ip for AWS to keep the fixed 172.20.0.9 for master_internal_ip
	i := api.Instance{
		ObjectMeta: api.ObjectMeta{
			UID:  phid.NewKubeInstance(),
			Name: *r1.Reservations[0].Instances[0].PrivateDnsName,
		},
		Spec: api.InstanceSpec{
			SKU: *r1.Reservations[0].Instances[0].InstanceType,
		},
		Status: api.InstanceStatus{
			ExternalID:    instanceID,
			ExternalPhase: *r1.Reservations[0].Instances[0].State.Name,
			ExternalIP:    *r1.Reservations[0].Instances[0].PublicIpAddress,
			InternalIP:    *r1.Reservations[0].Instances[0].PrivateIpAddress,
		},
	}
	/*
		// The low byte represents the state. The high byte is an opaque internal value
		// and should be ignored.
		//
		//    0 : pending
		//    16 : running
		//    32 : shutting-down
		//    48 : terminated
		//    64 : stopping
		//    80 : stopped
	*/
	if i.Status.ExternalPhase == "terminated" {
		i.Status.Phase = api.InstancePhaseDeleted
	} else {
		i.Status.Phase = api.InstancePhaseReady
	}
	return &i, nil
}

func (cm *ClusterManager) allocateElasticIp() (string, error) {
	r1, err := cm.conn.ec2.AllocateAddress(&_ec2.AllocateAddressInput{
		Domain: types.StringP("vpc"),
	})
	cloud.Logger(cm.ctx).Debug("Allocated elastic IP", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Elastic IP %v allocated", *r1.PublicIp)
	time.Sleep(5 * time.Second)
	return *r1.PublicIp, nil
}

func (cm *ClusterManager) assignIPToInstance(instanceID string) error {
	r1, err := cm.conn.ec2.DescribeAddresses(&_ec2.DescribeAddressesInput{
		PublicIps: []*string{types.StringP(cm.cluster.Spec.MasterReservedIP)},
	})
	cloud.Logger(cm.ctx).Debug("Retrieved allocation ID for elastic IP", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Found allocation id %v for elastic IP %v", r1.Addresses[0].AllocationId, cm.cluster.Spec.MasterReservedIP)
	time.Sleep(1 * time.Minute)

	r2, err := cm.conn.ec2.AssociateAddress(&_ec2.AssociateAddressInput{
		InstanceId:   types.StringP(instanceID),
		AllocationId: r1.Addresses[0].AllocationId,
	})
	cloud.Logger(cm.ctx).Debug("Attached IP to instance", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("IP %v attached to instance %v", cm.cluster.Spec.MasterReservedIP, instanceID)
	return nil
}

func (cm *ClusterManager) createLaunchConfiguration(name, sku string) error {
	// script := cm.RenderStartupScript(cm.cluster, sku, api.RoleKubernetesPool)
	script, err := cloud.RenderStartupScript(cm.ctx, cm.cluster, api.RoleKubernetesPool)
	if err != nil {
		return err
	}
	configuration := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  types.StringP(name),
		AssociatePublicIpAddress: types.BoolP(cm.cluster.Spec.EnableNodePublicIP),
		// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html
		BlockDeviceMappings: []*autoscaling.BlockDeviceMapping{
			// NODE_BLOCK_DEVICE_MAPPINGS
			{
				// https://github.com/appscode/kubernetes/blob/55d9dec8eb5eb02e1301045b7b81bbac689c86a1/cluster/aws/util.sh#L397
				DeviceName: types.StringP(cm.cluster.Spec.RootDeviceName),
				Ebs: &autoscaling.Ebs{
					DeleteOnTermination: types.TrueP(),
					VolumeSize:          types.Int64P(cm.cluster.Spec.NodeDiskSize),
					VolumeType:          types.StringP(cm.cluster.Spec.NodeDiskType),
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
		IamInstanceProfile: types.StringP(cm.cluster.Spec.IAMProfileNode),
		ImageId:            types.StringP(cm.cluster.Spec.InstanceImage),
		InstanceType:       types.StringP(sku),
		KeyName:            types.StringP(cm.cluster.Spec.SSHKeyExternalID),
		SecurityGroups: []*string{
			types.StringP(cm.cluster.Spec.NodeSGId),
		},
		UserData: types.StringP(base64.StdEncoding.EncodeToString([]byte(script))),
	}
	r1, err := cm.conn.autoscale.CreateLaunchConfiguration(configuration)
	cloud.Logger(cm.ctx).Debug("Created node configuration", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Info("Node configuration created assuming node public ip is enabled")
	return nil
}

func (cm *ClusterManager) createAutoScalingGroup(name, launchConfig string, count int64) error {
	r2, err := cm.conn.autoscale.CreateAutoScalingGroup(&autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: types.StringP(name),
		MaxSize:              types.Int64P(count),
		MinSize:              types.Int64P(count),
		DesiredCapacity:      types.Int64P(count),
		AvailabilityZones: []*string{
			types.StringP(cm.cluster.Spec.Zone),
		},
		LaunchConfigurationName: types.StringP(launchConfig),
		Tags: []*autoscaling.Tag{
			{
				Key:          types.StringP("Name"),
				ResourceId:   types.StringP(name),
				ResourceType: types.StringP("auto-scaling-group"),
				Value:        types.StringP(name), // node instance prefix LN_1042
			},
			{
				Key:          types.StringP("Role"),
				ResourceId:   types.StringP(name),
				ResourceType: types.StringP("auto-scaling-group"),
				Value:        types.StringP(cm.cluster.Name + "-node"),
			},
			{
				Key:          types.StringP("KubernetesCluster"),
				ResourceId:   types.StringP(name),
				ResourceType: types.StringP("auto-scaling-group"),
				Value:        types.StringP(cm.cluster.Name),
			},
		},
		VPCZoneIdentifier: types.StringP(cm.cluster.Spec.SubnetId),
	})
	cloud.Logger(cm.ctx).Debug("Created autoscaling group", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Autoscaling group %v created", name)
	return nil
}

func (cm *ClusterManager) detectMaster() error {
	masterID, err := cm.getInstanceIDFromName(cm.cluster.Spec.KubernetesMasterName)
	if masterID == "" {
		cloud.Logger(cm.ctx).Info("Could not detect Kubernetes master node.  Make sure you've launched a cluster with appctl.")
		//os.Exit(0)
	}
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterIP, _, err := cm.getInstancePublicIP(masterID)
	if masterIP == "" {
		cloud.Logger(cm.ctx).Info("Could not detect Kubernetes master node IP.  Make sure you've launched a cluster with appctl")
		os.Exit(0)
	}
	cloud.Logger(cm.ctx).Infof("Using master: %v (external IP: %v)", cm.cluster.Spec.KubernetesMasterName, masterIP)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *ClusterManager) getInstanceIDFromName(tagName string) (string, error) {
	r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:Name"),
				Values: []*string{
					types.StringP(tagName),
				},
			},
			{
				Name: types.StringP("instance-state-name"),
				Values: []*string{
					types.StringP("running"),
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
	cloud.Logger(cm.ctx).Debug("Retrieved instace via name", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if r1.Reservations != nil && r1.Reservations[0].Instances != nil {
		return *r1.Reservations[0].Instances[0].InstanceId, nil
	}
	return "", nil
}
