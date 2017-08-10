package aws

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"
	// "github.com/appscode/pharmer/templates"
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/phid"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
	// "github.com/appscode/pharmer/templates"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_iam "github.com/aws/aws-sdk-go/service/iam"
)

const (
	preTagDelay = 5 * time.Second
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = lib.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn, err = NewConnector(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Save()

	defer func(releaseReservedIp bool) {
		if cm.ctx.Status == storage.KubernetesStatus_Pending {
			cm.ctx.Status = storage.KubernetesStatus_Failing
		}
		cm.ctx.Save()
		cm.ins.Save()
		cm.ctx.Logger().Infof("Cluster %v is %v", cm.ctx.Name, cm.ctx.Status)
		if cm.ctx.Status != storage.KubernetesStatus_Ready {
			cm.ctx.Logger().Infof("Cluster %v is deleting", cm.ctx.Name)
			cm.delete(&proto.ClusterDeleteRequest{
				Name:              cm.ctx.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.ctx.MasterReservedIP == "auto")

	if err = cm.conn.detectJessieImage(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.InstanceImage = cm.conn.ctx.InstanceImage
	cm.ctx.RootDeviceName = cm.conn.ctx.RootDeviceName
	fmt.Println(cm.ctx.InstanceImage, cm.ctx.RootDeviceName, "---------------*********")

	if err = cm.ensureIAMProfile(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.importPublicKey(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupVpc(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.createDHCPOptionSet(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupSubnet(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupInternetGateway(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupRouteTable(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.setupSecurityGroups(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterInstance, err := cm.startMaster()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	for _, ng := range req.NodeGroups {
		igm := &InstanceGroupManager{
			cm: cm,
			instance: lib.Instance{
				Type: lib.InstanceType{
					ContextVersion: cm.ctx.ContextVersion,
					Sku:            ng.Sku,

					Master:       false,
					SpotInstance: false,
				},
				Stats: lib.GroupStats{
					Count: ng.Count,
				},
			},
		}
		igm.AdjustInstanceGroup()
	}

	cm.ctx.Logger().Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := lib.EnsureDnsIPLookup(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// wait for nodes to start
	if err := lib.ProbeKubeAPI(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = lib.CheckComponentStatuses(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = lib.WaitForReadyNodes(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------------------------------------------------
	cm.ctx.Logger().Info("Listing autoscaling groups")
	groups := make([]*string, 0)
	for _, ng := range req.NodeGroups {
		groups = append(groups, types.StringP(cm.namer.AutoScalingGroupName(ng.Sku)))
	}
	r2, err := cm.conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groups,
	})
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	fmt.Println(r2)
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	for _, group := range r2.AutoScalingGroups {
		for _, instance := range group.Instances {
			ki, err := cm.newKubeInstance(*instance.InstanceId)
			ki.Role = system.RoleKubernetesPool
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
	cm.ctx.Status = storage.KubernetesStatus_Ready
	return nil
}

func (cm *clusterManager) ensureIAMProfile() error {
	r1, _ := cm.conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &cm.ctx.IAMProfileMaster})
	if r1.InstanceProfile == nil {
		err := cm.createIAMProfile(cm.ctx.IAMProfileMaster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Infof("Master instance profile %v created", cm.ctx.IAMProfileMaster)
	}
	r2, _ := cm.conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &cm.ctx.IAMProfileNode})
	if r2.InstanceProfile == nil {
		err := cm.createIAMProfile(cm.ctx.IAMProfileNode)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Infof("Node instance profile %v created", cm.ctx.IAMProfileNode)
	}
	return nil
}

func (cm *clusterManager) createIAMProfile(key string) error {
	//rootDir := "kubernetes/aws/iam/"
	role := "" // TODO(tamal); FixIt!  templates.AssetText(rootDir + key + "-role.json")
	r1, err := cm.conn.iam.CreateRole(&_iam.CreateRoleInput{
		RoleName:                 &key,
		AssumeRolePolicyDocument: &role,
	})
	cm.ctx.Logger().Debug("Created IAM role", r1, err)
	cm.ctx.Logger().Infof("IAM role %v created", key)
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
	cm.ctx.Logger().Debug("Created IAM role-policy", r2, err)
	cm.ctx.Logger().Infof("IAM role-policy %v created", key)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	r3, err := cm.conn.iam.CreateInstanceProfile(&_iam.CreateInstanceProfileInput{
		InstanceProfileName: &key,
	})
	cm.ctx.Logger().Debug("Created IAM instance-policy", r3, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("IAM instance-policy %v created", key)

	r4, err := cm.conn.iam.AddRoleToInstanceProfile(&_iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: &key,
		RoleName:            &key,
	})
	cm.ctx.Logger().Debug("Added IAM role to instance-policy", r4, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("IAM role %v added to instance-policy %v", key, key)
	return nil
}

func (cm *clusterManager) importPublicKey() error {
	resp, err := cm.conn.ec2.ImportKeyPair(&_ec2.ImportKeyPairInput{
		KeyName:           types.StringP(cm.ctx.SSHKeyExternalID),
		PublicKeyMaterial: cm.ctx.SSHKey.PublicKey,
	})
	cm.ctx.Logger().Debug("Imported SSH key", resp, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// TODO ignore "InvalidKeyPair.Duplicate" error
	if err != nil {
		cm.ctx.Logger().Info("Error importing public key", resp, err)
		//os.Exit(1)
		return errors.FromErr(err).WithContext(cm.ctx).Err()

	}
	cm.ctx.Logger().Infof("SSH key with (AWS) fingerprint %v imported", cm.ctx.SSHKey.AwsFingerprint)

	return nil
}

func (cm *clusterManager) setupVpc() error {
	cm.ctx.Logger().Infof("Checking VPC tagged with %v", cm.ctx.Name)
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
					types.StringP(cm.ctx.Name), // Tag by Name or PHID?
				},
			},
		},
	})
	cm.ctx.Logger().Debug("VPC described", r1, err)
	if len(r1.Vpcs) > 1 {
		cm.ctx.VpcId = *r1.Vpcs[0].VpcId
		cm.ctx.Logger().Infof("VPC %v found", cm.ctx.VpcId)
	}

	cm.ctx.Logger().Info("No VPC found, creating new VPC")
	r2, err := cm.conn.ec2.CreateVpc(&_ec2.CreateVpcInput{
		CidrBlock: types.StringP(cm.ctx.VpcCidr),
	})
	cm.ctx.Logger().Debug("VPC created", r2, err)
	//errorutil.EOE(err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("VPC %v created", *r2.Vpc.VpcId)
	cm.ctx.VpcId = *r2.Vpc.VpcId

	r3, err := cm.conn.ec2.ModifyVpcAttribute(&_ec2.ModifyVpcAttributeInput{
		VpcId: types.StringP(cm.ctx.VpcId),
		EnableDnsSupport: &_ec2.AttributeBooleanValue{
			Value: types.TrueP(),
		},
	})
	cm.ctx.Logger().Debug("DNS support enabled", r3, err)
	cm.ctx.Logger().Infof("Enabled DNS support for VPCID %v", cm.ctx.VpcId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	r4, err := cm.conn.ec2.ModifyVpcAttribute(&_ec2.ModifyVpcAttributeInput{
		VpcId: types.StringP(cm.ctx.VpcId),
		EnableDnsHostnames: &_ec2.AttributeBooleanValue{
			Value: types.TrueP(),
		},
	})
	cm.ctx.Logger().Debug("DNS hostnames enabled", r4, err)
	cm.ctx.Logger().Infof("Enabled DNS hostnames for VPCID %v", cm.ctx.VpcId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	time.Sleep(preTagDelay)
	cm.addTag(cm.ctx.VpcId, "Name", cm.namer.VPCName())
	cm.addTag(cm.ctx.VpcId, "KubernetesCluster", cm.ctx.Name)
	return nil
}

func (cm *clusterManager) addTag(id string, key string, value string) error {
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
	cm.ctx.Logger().Debug("Added tag ", resp, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Added tag %v:%v to id %v", key, value, id)
	return nil
}

func (cm *clusterManager) createDHCPOptionSet() error {
	optionSetDomain := fmt.Sprintf("%v.compute.internal", cm.ctx.Region)
	if cm.ctx.Region == "us-east-1" {
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
	cm.ctx.Logger().Debug("Created DHCP options ", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("DHCP options created with id %v", *r1.DhcpOptions.DhcpOptionsId)
	cm.ctx.DHCPOptionsId = *r1.DhcpOptions.DhcpOptionsId

	time.Sleep(preTagDelay)
	cm.addTag(cm.ctx.DHCPOptionsId, "Name", cm.namer.DHCPOptionsName())
	cm.addTag(cm.ctx.DHCPOptionsId, "KubernetesCluster", cm.ctx.Name)

	r2, err := cm.conn.ec2.AssociateDhcpOptions(&_ec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: types.StringP(cm.ctx.DHCPOptionsId),
		VpcId:         types.StringP(cm.ctx.VpcId),
	})
	cm.ctx.Logger().Debug("Associated DHCP options ", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("DHCP options %v associated with %v", cm.ctx.DHCPOptionsId, cm.ctx.VpcId)

	return nil
}

func (cm *clusterManager) setupSubnet() error {
	cm.ctx.Logger().Info("Checking for existing subnet")
	r1, err := cm.conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.ctx.Name),
				},
			},
			{
				Name: types.StringP("availabilityZone"),
				Values: []*string{
					types.StringP(cm.ctx.Zone),
				},
			},
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cm.ctx.VpcId),
				},
			},
		},
	})
	cm.ctx.Logger().Debug("Retrieved subnet", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if len(r1.Subnets) == 0 {
		cm.ctx.Logger().Info("No subnet found, creating new subnet")
		r2, err := cm.conn.ec2.CreateSubnet(&_ec2.CreateSubnetInput{
			CidrBlock:        types.StringP(cm.ctx.SubnetCidr),
			VpcId:            types.StringP(cm.ctx.VpcId),
			AvailabilityZone: types.StringP(cm.ctx.Zone),
		})
		cm.ctx.Logger().Debug("Created subnet", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Infof("Subnet %v created", *r2.Subnet.SubnetId)
		cm.ctx.SubnetId = *r2.Subnet.SubnetId

		time.Sleep(preTagDelay)
		cm.addTag(cm.ctx.SubnetId, "KubernetesCluster", cm.ctx.Name)

	} else {
		cm.ctx.SubnetId = *r1.Subnets[0].SubnetId
		existingCIDR := *r1.Subnets[0].CidrBlock
		cm.ctx.Logger().Infof("Subnet %v found with CIDR %v", cm.ctx.SubnetId, existingCIDR)

		cm.ctx.Logger().Infof("Retrieving VPC %v", cm.ctx.VpcId)
		r3, err := cm.conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
			VpcIds: []*string{types.StringP(cm.ctx.VpcId)},
		})
		cm.ctx.Logger().Debug("Retrieved VPC", r3, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		octets := strings.Split(*r3.Vpcs[0].CidrBlock, ".")
		cm.ctx.VpcCidrBase = octets[0] + "." + octets[1]
		cm.ctx.MasterInternalIP = cm.ctx.VpcCidrBase + ".0" + cm.ctx.MasterIPSuffix
		cm.ctx.Logger().Infof("Assuming MASTER_INTERNAL_IP=%v", cm.ctx.MasterInternalIP)
	}
	return nil
}

func (cm *clusterManager) setupInternetGateway() error {
	cm.ctx.Logger().Infof("Checking IGW with attached VPCID %v", cm.ctx.VpcId)
	r1, err := cm.conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("attachment.vpc-id"),
				Values: []*string{
					types.StringP(cm.ctx.VpcId),
				},
			},
		},
	})
	cm.ctx.Logger().Debug("Retrieved IGW", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if len(r1.InternetGateways) == 0 {
		cm.ctx.Logger().Info("No IGW found, creating new IGW")
		r2, err := cm.conn.ec2.CreateInternetGateway(&_ec2.CreateInternetGatewayInput{})
		cm.ctx.Logger().Debug("Created IGW", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.IGWId = *r2.InternetGateway.InternetGatewayId
		time.Sleep(preTagDelay)
		cm.ctx.Logger().Infof("IGW %v created", cm.ctx.IGWId)

		r3, err := cm.conn.ec2.AttachInternetGateway(&_ec2.AttachInternetGatewayInput{
			InternetGatewayId: types.StringP(cm.ctx.IGWId),
			VpcId:             types.StringP(cm.ctx.VpcId),
		})
		cm.ctx.Logger().Debug("Attached IGW to VPC", r3, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Infof("Attached IGW %v to VPCID %v", cm.ctx.IGWId, cm.ctx.VpcId)

		cm.addTag(cm.ctx.IGWId, "Name", cm.namer.InternetGatewayName())
		cm.addTag(cm.ctx.IGWId, "KubernetesCluster", cm.ctx.Name)
	} else {
		cm.ctx.IGWId = *r1.InternetGateways[0].InternetGatewayId
		cm.ctx.Logger().Infof("IGW %v found", cm.ctx.IGWId)
	}
	return nil
}

func (cm *clusterManager) setupRouteTable() error {
	cm.ctx.Logger().Infof("Checking route table for VPCID %v", cm.ctx.VpcId)
	r1, err := cm.conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cm.ctx.VpcId),
				},
			},
			{
				Name: types.StringP("tag:KubernetesCluster"),
				Values: []*string{
					types.StringP(cm.ctx.Name),
				},
			},
		},
	})
	cm.ctx.Logger().Debug("Attached IGW to VPC", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if len(r1.RouteTables) == 0 {
		cm.ctx.Logger().Infof("No route table found for VPCID %v, creating new route table", cm.ctx.VpcId)
		r2, err := cm.conn.ec2.CreateRouteTable(&_ec2.CreateRouteTableInput{
			VpcId: types.StringP(cm.ctx.VpcId),
		})
		cm.ctx.Logger().Debug("Created route table", r2, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		cm.ctx.RouteTableId = *r2.RouteTable.RouteTableId
		cm.ctx.Logger().Infof("Route table %v created", cm.ctx.RouteTableId)
		time.Sleep(preTagDelay)
		cm.addTag(cm.ctx.RouteTableId, "KubernetesCluster", cm.ctx.Name)

	} else {
		cm.ctx.RouteTableId = *r1.RouteTables[0].RouteTableId
		cm.ctx.Logger().Infof("Route table %v found", cm.ctx.RouteTableId)
	}

	r3, err := cm.conn.ec2.AssociateRouteTable(&_ec2.AssociateRouteTableInput{
		RouteTableId: types.StringP(cm.ctx.RouteTableId),
		SubnetId:     types.StringP(cm.ctx.SubnetId),
	})
	cm.ctx.Logger().Debug("Associating route table to subnet", r3, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Route table %v associated to subnet %v", cm.ctx.RouteTableId, cm.ctx.SubnetId)

	r4, err := cm.conn.ec2.CreateRoute(&_ec2.CreateRouteInput{
		RouteTableId:         types.StringP(cm.ctx.RouteTableId),
		DestinationCidrBlock: types.StringP("0.0.0.0/0"),
		GatewayId:            types.StringP(cm.ctx.IGWId),
	})
	cm.ctx.Logger().Debug("Added route to route table", r4, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Route added to route table %v", cm.ctx.RouteTableId)
	return nil
}

func (cm *clusterManager) setupSecurityGroups() error {
	var ok bool
	var err error
	if cm.ctx.MasterSGId, ok, err = cm.getSecurityGroupId(cm.ctx.MasterSGName); !ok {
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.createSecurityGroup(cm.ctx.MasterSGName, "Kubernetes security group applied to master instance")
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Infof("Master security group %v created", cm.ctx.MasterSGName)
	}
	if cm.ctx.NodeSGId, ok, err = cm.getSecurityGroupId(cm.ctx.NodeSGName); !ok {
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.createSecurityGroup(cm.ctx.NodeSGName, "Kubernetes security group applied to node instances")
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Infof("Naster security group %v created", cm.ctx.NodeSGName)
	}

	err = cm.detectSecurityGroups()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Logger().Info("Masters can talk to master")
	err = cm.autohrizeIngressBySGID(cm.ctx.MasterSGId, cm.ctx.MasterSGId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Logger().Info("Nodes can talk to nodes")
	err = cm.autohrizeIngressBySGID(cm.ctx.NodeSGId, cm.ctx.NodeSGId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Logger().Info("Masters and nodes can talk to each other")
	err = cm.autohrizeIngressBySGID(cm.ctx.MasterSGId, cm.ctx.NodeSGId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.autohrizeIngressBySGID(cm.ctx.NodeSGId, cm.ctx.MasterSGId)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// TODO(justinsb): Would be fairly easy to replace 0.0.0.0/0 in these rules

	cm.ctx.Logger().Info("SSH is opened to the world")
	err = cm.autohrizeIngressByPort(cm.ctx.MasterSGId, 22)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.autohrizeIngressByPort(cm.ctx.NodeSGId, 22)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Logger().Info("HTTPS to the master is allowed (for API access)")
	err = cm.autohrizeIngressByPort(cm.ctx.MasterSGId, 443)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.autohrizeIngressByPort(cm.ctx.MasterSGId, 6443)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) getSecurityGroupId(groupName string) (string, bool, error) {
	cm.ctx.Logger().Infof("Checking security group %v", groupName)
	r1, err := cm.conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(cm.ctx.VpcId),
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
					types.StringP(cm.ctx.Name),
				},
			},
		},
	})
	cm.ctx.Logger().Debug("Retrieved security group", r1, err)
	if err != nil {
		return "", false, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if len(r1.SecurityGroups) == 0 {
		cm.ctx.Logger().Infof("No security group %v found", groupName)
		return "", false, nil
	}
	cm.ctx.Logger().Infof("Security group %v found", groupName)
	return *r1.SecurityGroups[0].GroupId, true, nil
}

func (cm *clusterManager) createSecurityGroup(groupName string, description string) error {
	cm.ctx.Logger().Infof("Creating security group %v", groupName)
	r2, err := cm.conn.ec2.CreateSecurityGroup(&_ec2.CreateSecurityGroupInput{
		GroupName:   types.StringP(groupName),
		Description: types.StringP(description),
		VpcId:       types.StringP(cm.ctx.VpcId),
	})
	cm.ctx.Logger().Debug("Created security group", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	time.Sleep(preTagDelay)
	err = cm.addTag(*r2.GroupId, "KubernetesCluster", cm.ctx.Name)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) detectSecurityGroups() error {
	var ok bool
	var err error
	if cm.ctx.MasterSGId == "" {
		if cm.ctx.MasterSGId, ok, err = cm.getSecurityGroupId(cm.ctx.MasterSGName); !ok {
			return errors.New("Could not detect Kubernetes master security group.  Make sure you've launched a cluster with appctl").WithContext(cm.ctx).Err()
		} else {
			cm.ctx.Logger().Infof("Master security group %v with id %v detected", cm.ctx.MasterSGName, cm.ctx.MasterSGId)
		}
	}
	if cm.ctx.NodeSGId == "" {
		if cm.ctx.NodeSGId, ok, err = cm.getSecurityGroupId(cm.ctx.NodeSGName); !ok {
			return errors.New("Could not detect Kubernetes node security group.  Make sure you've launched a cluster with appctl").WithContext(cm.ctx).Err()
		} else {
			cm.ctx.Logger().Infof("Node security group %v with id %v detected", cm.ctx.NodeSGName, cm.ctx.NodeSGId)
		}
	}
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) autohrizeIngressBySGID(groupID string, srcGroup string) error {
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
	cm.ctx.Logger().Debug("Authorized ingress", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Ingress authorized into SG %v from SG %v", groupID, srcGroup)
	return nil
}

func (cm *clusterManager) autohrizeIngressByPort(groupID string, port int64) error {
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
	cm.ctx.Logger().Debug("Authorized ingress", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Authorized ingress into SG %v via port %v", groupID, port)
	return nil
}

//
// -------------------------------------
//
func (cm *clusterManager) startMaster() (*contexts.KubernetesInstance, error) {
	var err error
	cm.ctx.MasterDiskId, err = cm.ensurePd(cm.namer.MasterPDName(), cm.ctx.MasterDiskType, cm.ctx.MasterDiskSize)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.reserveIP()
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	lib.GenClusterCerts(cm.ctx)
	cm.ctx.Save() // needed for master start-up config
	cm.UploadStartupConfig()

	masterInstanceID, err := cm.createMasterInstance(cm.ctx.KubernetesMasterName, system.RoleKubernetesMaster)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Info("Waiting for master instance to be ready")
	// We are not able to add an elastic ip, a route or volume to the instance until that instance is in "running" state.
	err = cm.waitForInstanceState(masterInstanceID, "running")
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Info("Master instance is ready")
	if cm.ctx.MasterReservedIP != "" {
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
	masterInstance.Role = system.RoleKubernetesMaster
	cm.ctx.MasterExternalIP = masterInstance.ExternalIP
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	err = lib.EnsureARecord(cm.ctx, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		return masterInstance, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.DetectApiServerURL()
	err = cm.ctx.Save() // needed for node start-up config to get master_internal_ip
	// This is a race between instance start and volume attachment.
	// There appears to be no way to start an AWS instance with a volume attached.
	// To work around this, we wait for volume to be ready in setup-master-pd.sh
	if err != nil {
		return masterInstance, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	r1, err := cm.conn.ec2.AttachVolume(&_ec2.AttachVolumeInput{
		VolumeId:   types.StringP(cm.ctx.MasterDiskId),
		Device:     types.StringP("/dev/sdb"),
		InstanceId: types.StringP(masterInstanceID),
	})
	cm.ctx.Logger().Debug("Attached persistent data volume to master", r1, err)
	if err != nil {
		return masterInstance, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Persistent data volume %v attatched to master", cm.ctx.MasterDiskId)

	time.Sleep(15 * time.Second)
	r2, err := cm.conn.ec2.CreateRoute(&_ec2.CreateRouteInput{
		RouteTableId:         types.StringP(cm.ctx.RouteTableId),
		DestinationCidrBlock: types.StringP(cm.ctx.MasterIPRange),
		InstanceId:           types.StringP(masterInstanceID),
	})
	cm.ctx.Logger().Debug("Created route to master", r2, err)
	if err != nil {
		return masterInstance, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Master route to route table %v for ip %v created", cm.ctx.RouteTableId, masterInstanceID)
	return masterInstance, nil
}

func (cm *clusterManager) ensurePd(name, diskType string, sizeGb int64) (string, error) {
	volumeId, err := cm.findPD(name)
	if err != nil {
		return volumeId, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if volumeId == "" {
		// name := cluster.ctx.KubernetesMasterName + "-pd"
		r1, err := cm.conn.ec2.CreateVolume(&_ec2.CreateVolumeInput{
			AvailabilityZone: &cm.ctx.Zone,
			VolumeType:       &diskType,
			Size:             types.Int64P(sizeGb),
		})
		cm.ctx.Logger().Debug("Created master pd", r1, err)
		if err != nil {
			return "", errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		volumeId = *r1.VolumeId
		cm.ctx.Logger().Infof("Master disk with size %vGB, type %v created", cm.ctx.MasterDiskSize, cm.ctx.MasterDiskType)

		time.Sleep(preTagDelay)
		err = cm.addTag(volumeId, "Name", name)
		if err != nil {
			return volumeId, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.addTag(volumeId, "KubernetesCluster", cm.ctx.Name)
		if err != nil {
			return volumeId, errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	return volumeId, nil
}

func (cm *clusterManager) findPD(name string) (string, error) {
	// name := cluster.ctx.KubernetesMasterName + "-pd"
	cm.ctx.Logger().Infof("Searching master pd %v", name)
	r1, err := cm.conn.ec2.DescribeVolumes(&_ec2.DescribeVolumesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("availability-zone"),
				Values: []*string{
					types.StringP(cm.ctx.Zone),
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
					types.StringP(cm.ctx.Name),
				},
			},
		},
	})
	cm.ctx.Logger().Debug("Retrieved master pd", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if len(r1.Volumes) > 0 {
		cm.ctx.Logger().Infof("Found master pd %v", name)
		return *r1.Volumes[0].VolumeId, nil
	}
	cm.ctx.Logger().Infof("Master pd %v not found", name)
	return "", nil
}

func (cm *clusterManager) reserveIP() error {
	// Check that MASTER_RESERVED_IP looks like an IPv4 address
	// if match, _ := regexp.MatchString("^[0-9]+.[0-9]+.[0-9]+.[0-9]+$", cluster.ctx.MasterReservedIP); !match {
	if cm.ctx.MasterReservedIP == "auto" {
		r1, err := cm.conn.ec2.AllocateAddress(&_ec2.AllocateAddressInput{
			Domain: types.StringP("vpc"),
		})
		cm.ctx.Logger().Debug("Allocated elastic IP", r1, err)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		time.Sleep(5 * time.Second)
		cm.ctx.MasterReservedIP = *r1.PublicIp
		cm.ctx.Logger().Infof("Elastic IP %v allocated", cm.ctx.MasterReservedIP)
	}
	return nil
}

func (cm *clusterManager) createMasterInstance(instanceName string, role string) (string, error) {
	kubeStarter := cm.RenderStartupScript(cm.ctx.NewScriptOptions(), cm.ctx.MasterSKU, role)
	req := &_ec2.RunInstancesInput{
		ImageId:  types.StringP(cm.ctx.InstanceImage),
		MaxCount: types.Int64P(1),
		MinCount: types.Int64P(1),
		// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html
		BlockDeviceMappings: []*_ec2.BlockDeviceMapping{
			// MASTER_BLOCK_DEVICE_MAPPINGS
			{
				// https://github.com/appscode/kubernetes/blob/55d9dec8eb5eb02e1301045b7b81bbac689c86a1/cluster/aws/util.sh#L397
				DeviceName: types.StringP(cm.ctx.RootDeviceName),
				Ebs: &_ec2.EbsBlockDevice{
					DeleteOnTermination: types.TrueP(),
					VolumeSize:          types.Int64P(cm.ctx.MasterDiskSize),
					VolumeType:          types.StringP(cm.ctx.MasterDiskType),
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
		IamInstanceProfile: &_ec2.IamInstanceProfileSpecification{
			Name: types.StringP(cm.ctx.IAMProfileMaster),
		},
		InstanceType: types.StringP(cm.ctx.MasterSKU),
		KeyName:      types.StringP(cm.ctx.SSHKeyExternalID),
		Monitoring: &_ec2.RunInstancesMonitoringEnabled{
			Enabled: types.TrueP(),
		},
		NetworkInterfaces: []*_ec2.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: types.TrueP(),
				DeleteOnTermination:      types.TrueP(),
				DeviceIndex:              types.Int64P(0),
				Groups: []*string{
					types.StringP(cm.ctx.MasterSGId),
				},
				PrivateIpAddresses: []*_ec2.PrivateIpAddressSpecification{
					{
						PrivateIpAddress: types.StringP(cm.ctx.MasterInternalIP),
						Primary:          types.TrueP(),
					},
				},
				SubnetId: types.StringP(cm.ctx.SubnetId),
			},
		},
		UserData: types.StringP(base64.StdEncoding.EncodeToString([]byte(kubeStarter))),
	}
	r1, err := cm.conn.ec2.RunInstances(req)
	cm.ctx.Logger().Debug("Created instance", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Instance %v created with role %v", instanceName, role)
	instanceID := *r1.Instances[0].InstanceId
	time.Sleep(preTagDelay)

	err = cm.addTag(instanceID, "Name", cm.ctx.KubernetesMasterName)
	if err != nil {
		return instanceID, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.addTag(instanceID, "Role", role)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.addTag(instanceID, "KubernetesCluster", cm.ctx.Name)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return instanceID, nil
}

func (cm *clusterManager) getInstancePublicIP(instanceID string) (string, bool, error) {
	r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		InstanceIds: []*string{types.StringP(instanceID)},
	})
	cm.ctx.Logger().Debug("Retrieved Public IP for Instance", r1, err)
	if err != nil {
		return "", false, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if r1.Reservations != nil && r1.Reservations[0].Instances != nil && r1.Reservations[0].Instances[0].NetworkInterfaces != nil {
		cm.ctx.Logger().Infof("Public ip for instance id %v retrieved", instanceID)
		return *r1.Reservations[0].Instances[0].NetworkInterfaces[0].Association.PublicIp, true, nil
	}
	return "", false, nil
}

func (cm *clusterManager) listInstances(groupName string) ([]*contexts.KubernetesInstance, error) {
	r2, err := cm.conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			types.StringP(groupName),
		},
	})
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	instances := make([]*contexts.KubernetesInstance, 0)
	for _, group := range r2.AutoScalingGroups {
		for _, instance := range group.Instances {
			ki, err := cm.newKubeInstance(*instance.InstanceId)
			if err != nil {
				return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			ki.Role = system.RoleKubernetesPool
			instances = append(instances, ki)
		}
	}
	return instances, nil
}
func (cm *clusterManager) newKubeInstance(instanceID string) (*contexts.KubernetesInstance, error) {
	r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		InstanceIds: []*string{types.StringP(instanceID)},
	})
	cm.ctx.Logger().Debug("Retrieved instance ", r1, err)
	if err != nil {
		return nil, lib.InstanceNotFound
	}

	// Don't reassign internal_ip for AWS to keep the fixed 172.20.0.9 for master_internal_ip
	i := contexts.KubernetesInstance{
		PHID:           phid.NewKubeInstance(),
		ExternalID:     instanceID,
		ExternalStatus: *r1.Reservations[0].Instances[0].State.Name,
		Name:           *r1.Reservations[0].Instances[0].PrivateDnsName,
		ExternalIP:     *r1.Reservations[0].Instances[0].PublicIpAddress,
		InternalIP:     *r1.Reservations[0].Instances[0].PrivateIpAddress,
		SKU:            *r1.Reservations[0].Instances[0].InstanceType,
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
	if i.ExternalStatus == "terminated" {
		i.Status = storage.KubernetesInstanceStatus_Deleted
	} else {
		i.Status = storage.KubernetesInstanceStatus_Ready
	}
	return &i, nil
}

func (cm *clusterManager) allocateElasticIp() (string, error) {
	r1, err := cm.conn.ec2.AllocateAddress(&_ec2.AllocateAddressInput{
		Domain: types.StringP("vpc"),
	})
	cm.ctx.Logger().Debug("Allocated elastic IP", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Elastic IP %v allocated", *r1.PublicIp)
	time.Sleep(5 * time.Second)
	return *r1.PublicIp, nil
}

func (cm *clusterManager) assignIPToInstance(instanceID string) error {
	r1, err := cm.conn.ec2.DescribeAddresses(&_ec2.DescribeAddressesInput{
		PublicIps: []*string{types.StringP(cm.ctx.MasterReservedIP)},
	})
	cm.ctx.Logger().Debug("Retrieved allocation ID for elastic IP", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Found allocation id %v for elastic IP %v", r1.Addresses[0].AllocationId, cm.ctx.MasterReservedIP)
	time.Sleep(1 * time.Minute)

	r2, err := cm.conn.ec2.AssociateAddress(&_ec2.AssociateAddressInput{
		InstanceId:   types.StringP(instanceID),
		AllocationId: r1.Addresses[0].AllocationId,
	})
	cm.ctx.Logger().Debug("Attached IP to instance", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("IP %v attached to instance %v", cm.ctx.MasterReservedIP, instanceID)
	return nil
}

func (cm *clusterManager) RenderStartupScript(opt *contexts.ScriptOptions, sku, role string) string {
	cmd := fmt.Sprintf(`/usr/local/bin/aws s3api get-object --bucket %v --key kubernetes/context/%v/startup-config/%v.yaml /tmp/role.yaml
CONFIG=$(cat /tmp/role.yaml)`, opt.BucketName, opt.ContextVersion, role)
	return lib.RenderKubeStarter(opt, sku, cmd)
}

func (cm *clusterManager) createLaunchConfiguration(name, sku string) error {
	script := cm.RenderStartupScript(cm.ctx.NewScriptOptions(), sku, system.RoleKubernetesPool)
	cm.UploadStartupConfig()
	configuration := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  types.StringP(name),
		AssociatePublicIpAddress: types.BoolP(cm.ctx.EnableNodePublicIP),
		// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html
		BlockDeviceMappings: []*autoscaling.BlockDeviceMapping{
			// NODE_BLOCK_DEVICE_MAPPINGS
			{
				// https://github.com/appscode/kubernetes/blob/55d9dec8eb5eb02e1301045b7b81bbac689c86a1/cluster/aws/util.sh#L397
				DeviceName: types.StringP(cm.ctx.RootDeviceName),
				Ebs: &autoscaling.Ebs{
					DeleteOnTermination: types.TrueP(),
					VolumeSize:          types.Int64P(cm.ctx.NodeDiskSize),
					VolumeType:          types.StringP(cm.ctx.NodeDiskType),
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
		IamInstanceProfile: types.StringP(cm.ctx.IAMProfileNode),
		ImageId:            types.StringP(cm.ctx.InstanceImage),
		InstanceType:       types.StringP(sku),
		KeyName:            types.StringP(cm.ctx.SSHKeyExternalID),
		SecurityGroups: []*string{
			types.StringP(cm.ctx.NodeSGId),
		},
		UserData: types.StringP(base64.StdEncoding.EncodeToString([]byte(script))),
	}
	r1, err := cm.conn.autoscale.CreateLaunchConfiguration(configuration)
	cm.ctx.Logger().Debug("Created node configuration", r1, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Info("Node configuration created assuming node public ip is enabled")
	return nil
}

func (cm *clusterManager) createAutoScalingGroup(name, launchConfig string, count int64) error {
	r2, err := cm.conn.autoscale.CreateAutoScalingGroup(&autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: types.StringP(name),
		MaxSize:              types.Int64P(count),
		MinSize:              types.Int64P(count),
		DesiredCapacity:      types.Int64P(count),
		AvailabilityZones: []*string{
			types.StringP(cm.ctx.Zone),
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
				Value:        types.StringP(cm.ctx.Name + "-node"),
			},
			{
				Key:          types.StringP("KubernetesCluster"),
				ResourceId:   types.StringP(name),
				ResourceType: types.StringP("auto-scaling-group"),
				Value:        types.StringP(cm.ctx.Name),
			},
		},
		VPCZoneIdentifier: types.StringP(cm.ctx.SubnetId),
	})
	cm.ctx.Logger().Debug("Created autoscaling group", r2, err)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Autoscaling group %v created", name)
	return nil
}

func (cm *clusterManager) detectMaster() error {
	masterID, err := cm.getInstanceIDFromName(cm.ctx.KubernetesMasterName)
	if masterID == "" {
		cm.ctx.Logger().Info("Could not detect Kubernetes master node.  Make sure you've launched a cluster with appctl.")
		//os.Exit(0)
	}
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterIP, _, err := cm.getInstancePublicIP(masterID)
	if masterIP == "" {
		cm.ctx.Logger().Info("Could not detect Kubernetes master node IP.  Make sure you've launched a cluster with appctl")
		os.Exit(0)
	}
	cm.ctx.Logger().Infof("Using master: %v (external IP: %v)", cm.ctx.KubernetesMasterName, masterIP)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) getInstanceIDFromName(tagName string) (string, error) {
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
					types.StringP(cm.ctx.Name),
				},
			},
		},
	})
	cm.ctx.Logger().Debug("Retrieved instace via name", r1, err)
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if r1.Reservations != nil && r1.Reservations[0].Instances != nil {
		return *r1.Reservations[0].Instances[0].InstanceId, nil
	}
	return "", nil
}
