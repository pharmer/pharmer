package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/appscode/go/context"
	stringutil "github.com/appscode/go/strings"
	. "github.com/appscode/go/types"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_elb "github.com/aws/aws-sdk-go/service/elb"
	_iam "github.com/aws/aws-sdk-go/service/iam"
	_ "github.com/aws/aws-sdk-go/service/lightsail"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	namer   namer

	ec2       *_ec2.EC2
	elb       *_elb.ELB
	iam       *_iam.IAM
	autoscale *autoscaling.AutoScaling
	s3        *_s3.S3
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.AWS{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	config := &_aws.Config{
		Region:      &cluster.Spec.Cloud.Region,
		Credentials: credentials.NewStaticCredentials(typed.AccessKeyID(), typed.SecretAccessKey(), ""),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}
	conn := cloudConnector{
		ctx:       ctx,
		cluster:   cluster,
		ec2:       _ec2.New(sess),
		elb:       _elb.New(sess),
		iam:       _iam.New(sess),
		autoscale: autoscaling.New(sess),
		s3:        _s3.New(sess),
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		return nil, errors.Errorf("credential %s does not have necessary authorization. Reason: %s", cluster.Spec.CredentialName, msg)
	}
	return &conn, nil
}

// Returns true if unauthorized
func (conn *cloudConnector) IsUnauthorized() (bool, string) {
	policies := make(map[string]string)
	var marker *string
	for {
		resp, err := conn.iam.ListPolicies(&_iam.ListPoliciesInput{
			MaxItems: Int64P(1000),
			Marker:   marker,
		})
		if err != nil {
			break
		}
		for _, p := range resp.Policies {
			policies[*p.PolicyName] = *p.Arn
		}
		if !_aws.BoolValue(resp.IsTruncated) {
			break
		}
		marker = resp.Marker
	}

	required := []string{
		"IAMFullAccess",
		"AmazonEC2FullAccess",
		"AmazonEC2ContainerRegistryFullAccess",
		"AmazonS3FullAccess",
		"AmazonRoute53FullAccess",
	}
	missing := make([]string, 0)
	for _, name := range required {
		if _, found := policies[name]; !found {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return false, "Credential missing required authorization: " + strings.Join(missing, ", ")
	}
	return true, ""
}

func (conn *cloudConnector) detectUbuntuImage() error {
	conn.cluster.Spec.Cloud.OS = "ubuntu"
	r1, err := conn.ec2.DescribeImages(&_ec2.DescribeImagesInput{
		Owners: []*string{StringP("099720109477")},
		Filters: []*_ec2.Filter{
			{
				Name: StringP("name"),
				Values: []*string{
					StringP("ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-20170619.1"),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	conn.cluster.Spec.Cloud.InstanceImage = *r1.Images[0].ImageId
	conn.cluster.Status.Cloud.AWS.RootDeviceName = *r1.Images[0].RootDeviceName
	Logger(conn.ctx).Infof("Ubuntu image with %v for %v detected", conn.cluster.Spec.Cloud.InstanceImage, conn.cluster.Status.Cloud.AWS.RootDeviceName)
	return nil
}

func (conn *cloudConnector) getIAMProfile() (bool, error) {
	r1, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.cluster.Spec.Cloud.AWS.IAMProfileMaster})
	if r1.InstanceProfile == nil {
		return false, err
	}
	r2, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.cluster.Spec.Cloud.AWS.IAMProfileNode})
	if r2.InstanceProfile == nil {
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) ensureIAMProfile() error {
	r1, _ := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.cluster.Spec.Cloud.AWS.IAMProfileMaster})
	if r1.InstanceProfile == nil {
		err := conn.createIAMProfile(api.RoleMaster, conn.cluster.Spec.Cloud.AWS.IAMProfileMaster)
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Master instance profile %v created", conn.cluster.Spec.Cloud.AWS.IAMProfileMaster)
	}
	r2, _ := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.cluster.Spec.Cloud.AWS.IAMProfileNode})
	if r2.InstanceProfile == nil {
		err := conn.createIAMProfile(api.RoleNode, conn.cluster.Spec.Cloud.AWS.IAMProfileNode)
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Node instance profile %v created", conn.cluster.Spec.Cloud.AWS.IAMProfileNode)
	}
	return nil
}

func (conn *cloudConnector) deleteIAMProfile() error {
	if err := conn.deleteRolePolicy(conn.cluster.Spec.Cloud.AWS.IAMProfileMaster); err != nil {
		Logger(conn.ctx).Infoln("Failed to delete IAM instance-policy ", conn.cluster.Spec.Cloud.AWS.IAMProfileMaster, err)
	}
	if err := conn.deleteRolePolicy(conn.cluster.Spec.Cloud.AWS.IAMProfileNode); err != nil {
		Logger(conn.ctx).Infoln("Failed to delete IAM instance-policy ", conn.cluster.Spec.Cloud.AWS.IAMProfileNode, err)
	}
	return nil
}

func (conn *cloudConnector) deleteRolePolicy(role string) error {
	if _, err := conn.iam.RemoveRoleFromInstanceProfile(&_iam.RemoveRoleFromInstanceProfileInput{
		InstanceProfileName: &role,
		RoleName:            &role,
	}); err != nil {
		Logger(conn.ctx).Infoln("Failed to remove role from instance profile", role, err)
	}

	if _, err := conn.iam.DeleteRolePolicy(&_iam.DeleteRolePolicyInput{
		PolicyName: &role,
		RoleName:   &role,
	}); err != nil {
		Logger(conn.ctx).Infoln("Failed to delete role policy", role, err)
	}
	if _, err := conn.iam.DeleteRole(&_iam.DeleteRoleInput{
		RoleName: &role,
	}); err != nil {
		Logger(conn.ctx).Infoln("Failed to delete role", role, err)
	}

	if _, err := conn.iam.DeleteInstanceProfile(&_iam.DeleteInstanceProfileInput{
		InstanceProfileName: &role,
	}); err != nil {
		Logger(conn.ctx).Infoln("Failed to delete instance profile", role, err)
	}
	return nil
}
func (conn *cloudConnector) createIAMProfile(role, key string) error {
	reqRole := &_iam.CreateRoleInput{RoleName: &key}
	if role == api.RoleMaster {
		reqRole.AssumeRolePolicyDocument = StringP(strings.TrimSpace(IAMMasterRole))
	} else {
		reqRole.AssumeRolePolicyDocument = StringP(strings.TrimSpace(IAMNodeRole))
	}
	r1, err := conn.iam.CreateRole(reqRole)
	Logger(conn.ctx).Debug("Created IAM role", r1, err)
	Logger(conn.ctx).Infof("IAM role %v created", key)
	if err != nil {
		return err
	}

	reqPolicy := &_iam.PutRolePolicyInput{
		RoleName:   &key,
		PolicyName: &key,
	}
	if role == api.RoleMaster {
		reqPolicy.PolicyDocument = StringP(strings.TrimSpace(IAMMasterPolicy))
	} else {
		reqPolicy.PolicyDocument = StringP(strings.TrimSpace(IAMNodePolicy))
	}
	r2, err := conn.iam.PutRolePolicy(reqPolicy)
	Logger(conn.ctx).Debug("Created IAM role-policy", r2, err)
	Logger(conn.ctx).Infof("IAM role-policy %v created", key)
	if err != nil {
		return err
	}

	r3, err := conn.iam.CreateInstanceProfile(&_iam.CreateInstanceProfileInput{
		InstanceProfileName: &key,
	})
	Logger(conn.ctx).Debug("Created IAM instance-policy", r3, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("IAM instance-policy %v created", key)

	r4, err := conn.iam.AddRoleToInstanceProfile(&_iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: &key,
		RoleName:            &key,
	})
	Logger(conn.ctx).Debug("Added IAM role to instance-policy", r4, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("IAM role %v added to instance-policy %v", key, key)
	return nil
}

func (conn *cloudConnector) getPublicKey() (bool, error) {
	resp, err := conn.ec2.DescribeKeyPairs(&_ec2.DescribeKeyPairsInput{
		KeyNames: StringPSlice([]string{conn.cluster.Spec.Cloud.SSHKeyName}),
	})
	if err != nil {
		return false, err
	}
	if len(resp.KeyPairs) > 0 {
		return true, nil
	}
	return false, nil
}

func (conn *cloudConnector) importPublicKey() error {
	resp, err := conn.ec2.ImportKeyPair(&_ec2.ImportKeyPairInput{
		KeyName:           StringP(conn.cluster.Spec.Cloud.SSHKeyName),
		PublicKeyMaterial: SSHKey(conn.ctx).PublicKey,
	})
	Logger(conn.ctx).Debug("Imported SSH key", resp, err)
	if err != nil {
		return err
	}
	// TODO ignore "InvalidKeyPair.Duplicate" error
	if err != nil {
		Logger(conn.ctx).Info("Error importing public key", resp, err)
		//os.Exit(1)
		return err

	}
	Logger(conn.ctx).Infof("SSH key with (AWS) fingerprint %v imported", SSHKey(conn.ctx).AwsFingerprint)

	return nil
}

func (conn *cloudConnector) findVPC() (bool, error) {
	r1, err := conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
		VpcIds: []*string{
			StringP(conn.cluster.Status.Cloud.AWS.VpcId),
		},
	})
	if err != nil {
		return false, err
	}
	return len(r1.Vpcs) > 0, nil
}

func (conn *cloudConnector) getVpc() (bool, error) {
	Logger(conn.ctx).Infof("Checking VPC tagged with %v", conn.cluster.Name)
	r1, err := conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("tag:Name"),
				Values: []*string{
					StringP(conn.namer.VPCName()),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name), // Tag by Name or PHID?
				},
			},
		},
	})
	Logger(conn.ctx).Debug("VPC described", r1, err)
	if len(r1.Vpcs) > 0 {
		conn.cluster.Status.Cloud.AWS.VpcId = *r1.Vpcs[0].VpcId
		Logger(conn.ctx).Infof("VPC %v found", conn.cluster.Status.Cloud.AWS.VpcId)
		return true, nil
	}

	return false, nil
}

func (conn *cloudConnector) setupVpc() error {
	Logger(conn.ctx).Info("No VPC found, creating new VPC")
	r2, err := conn.ec2.CreateVpc(&_ec2.CreateVpcInput{
		CidrBlock: StringP(conn.cluster.Spec.Cloud.AWS.VpcCIDR),
	})
	Logger(conn.ctx).Debug("VPC created", r2, err)
	//errorutil.EOE(err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("VPC %v created", *r2.Vpc.VpcId)
	conn.cluster.Status.Cloud.AWS.VpcId = *r2.Vpc.VpcId

	r3, err := conn.ec2.ModifyVpcAttribute(&_ec2.ModifyVpcAttributeInput{
		VpcId: StringP(conn.cluster.Status.Cloud.AWS.VpcId),
		EnableDnsSupport: &_ec2.AttributeBooleanValue{
			Value: TrueP(),
		},
	})
	Logger(conn.ctx).Debug("DNS support enabled", r3, err)
	Logger(conn.ctx).Infof("Enabled DNS support for VPCID %v", conn.cluster.Status.Cloud.AWS.VpcId)
	if err != nil {
		return err
	}

	r4, err := conn.ec2.ModifyVpcAttribute(&_ec2.ModifyVpcAttributeInput{
		VpcId: StringP(conn.cluster.Status.Cloud.AWS.VpcId),
		EnableDnsHostnames: &_ec2.AttributeBooleanValue{
			Value: TrueP(),
		},
	})
	Logger(conn.ctx).Debug("DNS hostnames enabled", r4, err)
	Logger(conn.ctx).Infof("Enabled DNS hostnames for VPCID %v", conn.cluster.Status.Cloud.AWS.VpcId)
	if err != nil {
		return err
	}

	time.Sleep(preTagDelay)
	conn.addTag(conn.cluster.Status.Cloud.AWS.VpcId, "Name", conn.namer.VPCName())
	conn.addTag(conn.cluster.Status.Cloud.AWS.VpcId, "KubernetesCluster", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) addTag(id string, key string, value string) error {
	resp, err := conn.ec2.CreateTags(&_ec2.CreateTagsInput{
		Resources: []*string{
			StringP(id),
		},
		Tags: []*_ec2.Tag{
			{
				Key:   StringP(key),
				Value: StringP(value),
			},
		},
	})
	Logger(conn.ctx).Debug("Added tag ", resp, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Added tag %v:%v to id %v", key, value, id)
	return nil
}

func (conn cloudConnector) getDHCPOptionSet() (bool, error) {
	r1, err := conn.ec2.DescribeDhcpOptions(&_ec2.DescribeDhcpOptionsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name), // Tag by Name or PHID?
				},
			},
		},
	})
	if err != nil {
		return false, err
	}
	if len(r1.DhcpOptions) > 0 {
		return true, nil
	}
	return false, nil
}

func (conn *cloudConnector) createDHCPOptionSet() error {
	optionSetDomain := fmt.Sprintf("%v.compute.internal", conn.cluster.Spec.Cloud.Region)
	if conn.cluster.Spec.Cloud.Region == "us-east-1" {
		optionSetDomain = "ec2.internal"
	}
	r1, err := conn.ec2.CreateDhcpOptions(&_ec2.CreateDhcpOptionsInput{
		DhcpConfigurations: []*_ec2.NewDhcpConfiguration{
			{
				Key:    StringP("domain-name"),
				Values: []*string{StringP(optionSetDomain)},
			},
			{
				Key:    StringP("domain-name-servers"),
				Values: []*string{StringP("AmazonProvidedDNS")},
			},
		},
	})
	Logger(conn.ctx).Debug("Created DHCP options ", r1, err)
	if err != nil {
		return err
	}

	Logger(conn.ctx).Infof("DHCP options created with id %v", *r1.DhcpOptions.DhcpOptionsId)
	conn.cluster.Status.Cloud.AWS.DHCPOptionsId = *r1.DhcpOptions.DhcpOptionsId

	time.Sleep(preTagDelay)
	conn.addTag(conn.cluster.Status.Cloud.AWS.DHCPOptionsId, "Name", conn.namer.DHCPOptionsName())
	conn.addTag(conn.cluster.Status.Cloud.AWS.DHCPOptionsId, "KubernetesCluster", conn.cluster.Name)

	r2, err := conn.ec2.AssociateDhcpOptions(&_ec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: StringP(conn.cluster.Status.Cloud.AWS.DHCPOptionsId),
		VpcId:         StringP(conn.cluster.Status.Cloud.AWS.VpcId),
	})
	Logger(conn.ctx).Debug("Associated DHCP options ", r2, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("DHCP options %v associated with %v", conn.cluster.Status.Cloud.AWS.DHCPOptionsId, conn.cluster.Status.Cloud.AWS.VpcId)

	return nil
}

func (conn *cloudConnector) getSubnet() (bool, error) {
	Logger(conn.ctx).Info("Checking for existing subnet")
	r1, err := conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
			{
				Name: StringP("availabilityZone"),
				Values: []*string{
					StringP(conn.cluster.Spec.Cloud.Zone),
				},
			},
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.VpcId),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Retrieved subnet", r1, err)
	if err != nil {
		return false, err
	}
	if len(r1.Subnets) == 0 {
		return false, errors.Errorf("No subnet found")
	}
	conn.cluster.Status.Cloud.AWS.SubnetId = *r1.Subnets[0].SubnetId
	existingCIDR := *r1.Subnets[0].CidrBlock
	Logger(conn.ctx).Infof("Subnet %v found with CIDR %v", conn.cluster.Status.Cloud.AWS.SubnetId, existingCIDR)

	Logger(conn.ctx).Infof("Retrieving VPC %v", conn.cluster.Status.Cloud.AWS.VpcId)
	r3, err := conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
		VpcIds: []*string{StringP(conn.cluster.Status.Cloud.AWS.VpcId)},
	})
	Logger(conn.ctx).Debug("Retrieved VPC", r3, err)
	if err != nil {
		return true, err
	}

	octets := strings.Split(*r3.Vpcs[0].CidrBlock, ".")
	conn.cluster.Spec.Cloud.AWS.VpcCIDRBase = octets[0] + "." + octets[1]
	conn.cluster.Spec.MasterInternalIP = conn.cluster.Spec.Cloud.AWS.VpcCIDRBase + ".0" + conn.cluster.Spec.Cloud.AWS.MasterIPSuffix
	Logger(conn.ctx).Infof("Assuming MASTER_INTERNAL_IP=%v", conn.cluster.Spec.MasterInternalIP)
	return true, nil

}

func (conn *cloudConnector) setupSubnet() error {
	Logger(conn.ctx).Info("No subnet found, creating new subnet")
	r2, err := conn.ec2.CreateSubnet(&_ec2.CreateSubnetInput{
		CidrBlock:        StringP(conn.cluster.Spec.Cloud.AWS.SubnetCIDR),
		VpcId:            StringP(conn.cluster.Status.Cloud.AWS.VpcId),
		AvailabilityZone: StringP(conn.cluster.Spec.Cloud.Zone),
	})
	Logger(conn.ctx).Debug("Created subnet", r2, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Subnet %v created", *r2.Subnet.SubnetId)
	conn.cluster.Status.Cloud.AWS.SubnetId = *r2.Subnet.SubnetId

	time.Sleep(preTagDelay)
	conn.addTag(conn.cluster.Status.Cloud.AWS.SubnetId, "KubernetesCluster", conn.cluster.Name)

	Logger(conn.ctx).Infof("Retrieving VPC %v", conn.cluster.Status.Cloud.AWS.VpcId)
	r3, err := conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
		VpcIds: []*string{StringP(conn.cluster.Status.Cloud.AWS.VpcId)},
	})
	Logger(conn.ctx).Debug("Retrieved VPC", r3, err)
	if err != nil {
		return err
	}

	octets := strings.Split(*r3.Vpcs[0].CidrBlock, ".")
	conn.cluster.Spec.Cloud.AWS.VpcCIDRBase = octets[0] + "." + octets[1]
	conn.cluster.Spec.MasterInternalIP = conn.cluster.Spec.Cloud.AWS.VpcCIDRBase + ".0" + conn.cluster.Spec.Cloud.AWS.MasterIPSuffix
	Logger(conn.ctx).Infof("Assuming MASTER_INTERNAL_IP=%v", conn.cluster.Spec.MasterInternalIP)
	return nil
}

func (conn *cloudConnector) getInternetGateway() (bool, error) {
	Logger(conn.ctx).Infof("Checking IGW with attached VPCID %v", conn.cluster.Status.Cloud.AWS.VpcId)
	r1, err := conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("attachment.vpc-id"),
				Values: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.VpcId),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Retrieved IGW", r1, err)
	if err != nil {
		return false, err
	}
	if len(r1.InternetGateways) == 0 {
		return false, errors.Errorf("IGW not found")
	}
	conn.cluster.Status.Cloud.AWS.IGWId = *r1.InternetGateways[0].InternetGatewayId
	Logger(conn.ctx).Infof("IGW %v found", conn.cluster.Status.Cloud.AWS.IGWId)
	return true, nil
}

func (conn *cloudConnector) setupInternetGateway() error {
	Logger(conn.ctx).Info("No IGW found, creating new IGW")
	r2, err := conn.ec2.CreateInternetGateway(&_ec2.CreateInternetGatewayInput{})
	Logger(conn.ctx).Debug("Created IGW", r2, err)
	if err != nil {
		return err
	}
	conn.cluster.Status.Cloud.AWS.IGWId = *r2.InternetGateway.InternetGatewayId
	time.Sleep(preTagDelay)
	Logger(conn.ctx).Infof("IGW %v created", conn.cluster.Status.Cloud.AWS.IGWId)

	r3, err := conn.ec2.AttachInternetGateway(&_ec2.AttachInternetGatewayInput{
		InternetGatewayId: StringP(conn.cluster.Status.Cloud.AWS.IGWId),
		VpcId:             StringP(conn.cluster.Status.Cloud.AWS.VpcId),
	})
	Logger(conn.ctx).Debug("Attached IGW to VPC", r3, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Attached IGW %v to VPCID %v", conn.cluster.Status.Cloud.AWS.IGWId, conn.cluster.Status.Cloud.AWS.VpcId)

	conn.addTag(conn.cluster.Status.Cloud.AWS.IGWId, "Name", conn.namer.InternetGatewayName())
	conn.addTag(conn.cluster.Status.Cloud.AWS.IGWId, "KubernetesCluster", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) getRouteTable() (bool, error) {
	Logger(conn.ctx).Infof("Checking route table for VPCID %v", conn.cluster.Status.Cloud.AWS.VpcId)
	r1, err := conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.VpcId),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Attached IGW to VPC", r1, err)
	if err != nil {
		return false, err
	}
	if len(r1.RouteTables) == 0 {
		return false, errors.Errorf("Route table not found")
	}
	conn.cluster.Status.Cloud.AWS.RouteTableId = *r1.RouteTables[0].RouteTableId
	Logger(conn.ctx).Infof("Route table %v found", conn.cluster.Status.Cloud.AWS.RouteTableId)
	return true, nil
}

func (conn *cloudConnector) setupRouteTable() error {
	Logger(conn.ctx).Infof("No route table found for VPCID %v, creating new route table", conn.cluster.Status.Cloud.AWS.VpcId)
	r2, err := conn.ec2.CreateRouteTable(&_ec2.CreateRouteTableInput{
		VpcId: StringP(conn.cluster.Status.Cloud.AWS.VpcId),
	})
	Logger(conn.ctx).Debug("Created route table", r2, err)
	if err != nil {
		return err
	}

	conn.cluster.Status.Cloud.AWS.RouteTableId = *r2.RouteTable.RouteTableId
	Logger(conn.ctx).Infof("Route table %v created", conn.cluster.Status.Cloud.AWS.RouteTableId)
	time.Sleep(preTagDelay)
	conn.addTag(conn.cluster.Status.Cloud.AWS.RouteTableId, "KubernetesCluster", conn.cluster.Name)

	r3, err := conn.ec2.AssociateRouteTable(&_ec2.AssociateRouteTableInput{
		RouteTableId: StringP(conn.cluster.Status.Cloud.AWS.RouteTableId),
		SubnetId:     StringP(conn.cluster.Status.Cloud.AWS.SubnetId),
	})
	Logger(conn.ctx).Debug("Associating route table to subnet", r3, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Route table %v associated to subnet %v", conn.cluster.Status.Cloud.AWS.RouteTableId, conn.cluster.Status.Cloud.AWS.SubnetId)

	r4, err := conn.ec2.CreateRoute(&_ec2.CreateRouteInput{
		RouteTableId:         StringP(conn.cluster.Status.Cloud.AWS.RouteTableId),
		DestinationCidrBlock: StringP("0.0.0.0/0"),
		GatewayId:            StringP(conn.cluster.Status.Cloud.AWS.IGWId),
	})
	Logger(conn.ctx).Debug("Added route to route table", r4, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Route added to route table %v", conn.cluster.Status.Cloud.AWS.RouteTableId)
	return nil
}

func (conn *cloudConnector) setupSecurityGroups() error {
	var ok bool
	var err error
	if conn.cluster.Status.Cloud.AWS.MasterSGId, ok, err = conn.getSecurityGroupId(conn.cluster.Spec.Cloud.AWS.MasterSGName); !ok {
		if err != nil {
			return err
		}
		err = conn.createSecurityGroup(conn.cluster.Spec.Cloud.AWS.MasterSGName, "Kubernetes security group applied to master instance")
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Master security group %v created", conn.cluster.Spec.Cloud.AWS.MasterSGName)
	}
	if conn.cluster.Status.Cloud.AWS.NodeSGId, ok, err = conn.getSecurityGroupId(conn.cluster.Spec.Cloud.AWS.NodeSGName); !ok {
		if err != nil {
			return err
		}
		err = conn.createSecurityGroup(conn.cluster.Spec.Cloud.AWS.NodeSGName, "Kubernetes security group applied to node instances")
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Node security group %v created", conn.cluster.Spec.Cloud.AWS.NodeSGName)
	}

	err = conn.detectSecurityGroups()
	if err != nil {
		return err
	}

	Logger(conn.ctx).Info("Masters can talk to master")
	err = conn.autohrizeIngressBySGID(conn.cluster.Status.Cloud.AWS.MasterSGId, conn.cluster.Status.Cloud.AWS.MasterSGId)
	if err != nil {
		return err
	}

	Logger(conn.ctx).Info("Nodes can talk to nodes")
	err = conn.autohrizeIngressBySGID(conn.cluster.Status.Cloud.AWS.NodeSGId, conn.cluster.Status.Cloud.AWS.NodeSGId)
	if err != nil {
		return err
	}

	Logger(conn.ctx).Info("Masters and nodes can talk to each other")
	err = conn.autohrizeIngressBySGID(conn.cluster.Status.Cloud.AWS.MasterSGId, conn.cluster.Status.Cloud.AWS.NodeSGId)
	if err != nil {
		return err
	}
	err = conn.autohrizeIngressBySGID(conn.cluster.Status.Cloud.AWS.NodeSGId, conn.cluster.Status.Cloud.AWS.MasterSGId)
	if err != nil {
		return err
	}

	// TODO(justinsb): Would be fairly easy to replace 0.0.0.0/0 in these rules

	Logger(conn.ctx).Info("SSH is opened to the world")
	err = conn.autohrizeIngressByPort(conn.cluster.Status.Cloud.AWS.MasterSGId, 22)
	if err != nil {
		return err
	}
	err = conn.autohrizeIngressByPort(conn.cluster.Status.Cloud.AWS.NodeSGId, 22)
	if err != nil {
		return err
	}

	Logger(conn.ctx).Info("HTTPS to the master is allowed (for API access)")
	err = conn.autohrizeIngressByPort(conn.cluster.Status.Cloud.AWS.MasterSGId, 443)
	if err != nil {
		return err
	}
	if conn.cluster.Spec.API.BindPort != 443 {
		err = conn.autohrizeIngressByPort(conn.cluster.Status.Cloud.AWS.MasterSGId, int64(conn.cluster.Spec.API.BindPort))
		if err != nil {
			return err
		}
	}
	return nil
}

func (conn *cloudConnector) getSecurityGroupId(groupName string) (string, bool, error) {
	Logger(conn.ctx).Infof("Checking security group %v", groupName)
	r1, err := conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.VpcId),
				},
			},
			{
				Name: StringP("group-name"),
				Values: []*string{
					StringP(groupName),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Retrieved security group", r1, err)
	if err != nil {
		return "", false, err
	}
	if len(r1.SecurityGroups) == 0 {
		Logger(conn.ctx).Infof("No security group %v found", groupName)
		return "", false, nil
	}
	Logger(conn.ctx).Infof("Security group %v found", groupName)
	return *r1.SecurityGroups[0].GroupId, true, nil
}

func (conn *cloudConnector) createSecurityGroup(groupName string, description string) error {
	Logger(conn.ctx).Infof("Creating security group %v", groupName)
	r2, err := conn.ec2.CreateSecurityGroup(&_ec2.CreateSecurityGroupInput{
		GroupName:   StringP(groupName),
		Description: StringP(description),
		VpcId:       StringP(conn.cluster.Status.Cloud.AWS.VpcId),
	})
	Logger(conn.ctx).Debug("Created security group", r2, err)
	if err != nil {
		return err
	}

	time.Sleep(preTagDelay)
	err = conn.addTag(*r2.GroupId, "KubernetesCluster", conn.cluster.Name)
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) detectSecurityGroups() error {
	var ok bool
	var err error
	if conn.cluster.Status.Cloud.AWS.MasterSGId == "" {
		if conn.cluster.Status.Cloud.AWS.MasterSGId, ok, err = conn.getSecurityGroupId(conn.cluster.Spec.Cloud.AWS.MasterSGName); !ok {
			return errors.Errorf("[%s] could not detect Kubernetes master security group.  Make sure you've launched a cluster with appctl", ID(conn.ctx))
		} else {
			Logger(conn.ctx).Infof("Master security group %v with id %v detected", conn.cluster.Spec.Cloud.AWS.MasterSGName, conn.cluster.Status.Cloud.AWS.MasterSGId)
		}
	}
	if conn.cluster.Status.Cloud.AWS.NodeSGId == "" {
		if conn.cluster.Status.Cloud.AWS.NodeSGId, ok, err = conn.getSecurityGroupId(conn.cluster.Spec.Cloud.AWS.NodeSGName); !ok {
			return errors.Errorf("[%s] could not detect Kubernetes node security group.  Make sure you've launched a cluster with appctl", ID(conn.ctx))
		} else {
			Logger(conn.ctx).Infof("Node security group %v with id %v detected", conn.cluster.Spec.Cloud.AWS.NodeSGName, conn.cluster.Status.Cloud.AWS.NodeSGId)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) autohrizeIngressBySGID(groupID string, srcGroup string) error {
	r1, err := conn.ec2.AuthorizeSecurityGroupIngress(&_ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: StringP(groupID),
		IpPermissions: []*_ec2.IpPermission{
			{
				IpProtocol: StringP("-1"),
				UserIdGroupPairs: []*_ec2.UserIdGroupPair{
					{
						GroupId: StringP(srcGroup),
					},
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Authorized ingress", r1, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Ingress authorized into SG %v from SG %v", groupID, srcGroup)
	return nil
}

func (conn *cloudConnector) autohrizeIngressByPort(groupID string, port int64) error {
	r1, err := conn.ec2.AuthorizeSecurityGroupIngress(&_ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: StringP(groupID),
		IpPermissions: []*_ec2.IpPermission{
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(port),
				IpRanges: []*_ec2.IpRange{
					{
						CidrIp: StringP("0.0.0.0/0"),
					},
				},
				ToPort: Int64P(port),
			},
		},
	})
	Logger(conn.ctx).Debug("Authorized ingress", r1, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Authorized ingress into SG %v via port %v", groupID, port)
	return nil
}

func (conn *cloudConnector) getMaster() (bool, error) {
	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name:   StringP("tag:Name"),
				Values: StringPSlice([]string{conn.namer.MasterName()}),
			},
			{
				Name:   StringP("tag:Role"),
				Values: StringPSlice([]string{"master"}),
			},
			{
				Name:   StringP("tag:KubernetesCluster"),
				Values: StringPSlice([]string{conn.cluster.Name}),
			},
			{
				Name:   StringP("instance-state-name"),
				Values: StringPSlice([]string{"running"}),
			},
		},
	})
	if err != nil {
		return false, err
	}
	if len(r1.Reservations) == 0 {
		return false, nil
	}
	fmt.Println(r1, err, "....................")
	return true, err
}

func (conn *cloudConnector) startMaster(name string, ng *api.NodeGroup) (*api.NodeInfo, error) {
	var err error
	// TODO: FixIt!
	masterDiskId, err := conn.ensurePd(conn.namer.MasterPDName(), ng.Spec.Template.Spec.DiskType, ng.Spec.Template.Spec.DiskSize)
	if err != nil {
		return nil, err
	}

	Store(conn.ctx).Clusters().UpdateStatus(conn.cluster) // needed for master start-up config

	masterInstanceID, err := conn.createMasterInstance(name, ng)
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Info("Waiting for master instance to be ready")
	// We are not able to add an elastic ip, a route or volume to the instance until that instance is in "running" state.
	err = conn.waitForInstanceState(masterInstanceID, "running")
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Info("Master instance is ready")
	if ng.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
		var reservedIP string
		if len(conn.cluster.Status.ReservedIPs) == 0 {
			if reservedIP, err = conn.createReserveIP(ng); err != nil {
				return nil, err
			}
		} else {
			reservedIP = conn.cluster.Status.ReservedIPs[0].IP
		}
		err = conn.assignIPToInstance(reservedIP, masterInstanceID)
		if err != nil {
			return nil, errors.Wrapf(err, "[%s] failed to assign ip", ID(conn.ctx))
		}
		conn.cluster.Status.APIAddresses = append(conn.cluster.Status.APIAddresses, core.NodeAddress{
			Type:    core.NodeExternalIP,
			Address: reservedIP,
		})
		conn.cluster.Status.ReservedIPs = append(conn.cluster.Status.ReservedIPs, api.ReservedIP{
			IP: reservedIP,
		})
	} else {
		rx, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
			InstanceIds: []*string{StringP(masterInstanceID)},
		})
		if err != nil {
			return nil, err
		}
		if *rx.Reservations[0].Instances[0].PublicIpAddress != "" {
			conn.cluster.Status.APIAddresses = append(conn.cluster.Status.APIAddresses, core.NodeAddress{
				Type:    core.NodeExternalIP,
				Address: *rx.Reservations[0].Instances[0].PublicIpAddress,
			})
		}
	}

	// load again to get IP address assigned
	r, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		InstanceIds: []*string{StringP(masterInstanceID)},
	})
	if err != nil {
		return nil, err
	}
	node := api.NodeInfo{
		Name:       *r.Reservations[0].Instances[0].PrivateDnsName,
		ExternalID: masterInstanceID,
	}

	// Don't reassign internal_ip for AWS to keep the fixed 172.20.0.9 for master_internal_ip
	var publicIP, privateIP string
	if *r.Reservations[0].Instances[0].State.Name != "running" {
		publicIP = ""
		privateIP = ""
	} else {
		publicIP = *r.Reservations[0].Instances[0].PublicIpAddress
		privateIP = *r.Reservations[0].Instances[0].PrivateIpAddress
	}
	node.PublicIP = publicIP
	node.PrivateIP = privateIP

	// TODO check setting master IP is set properly
	//	masterInstance, err := conn.newKubeInstance(masterInstanceID) // sets external IP

	_, err = Store(conn.ctx).Clusters().UpdateStatus(conn.cluster) // needed for node start-up config to get master_internal_ip
	// This is a race between instance start and volume attachment.
	// There appears to be no way to start an AWS instance with a volume attached.
	// To work around this, we wait for volume to be ready in setup-master-pd.sh
	if err != nil {
		return &node, err
	}

	r1, err := conn.ec2.AttachVolume(&_ec2.AttachVolumeInput{
		VolumeId:   StringP(masterDiskId),
		Device:     StringP("/dev/sdb"),
		InstanceId: StringP(masterInstanceID),
	})
	Logger(conn.ctx).Debug("Attached persistent data volume to master", r1, err)
	if err != nil {
		return &node, err
	}
	Logger(conn.ctx).Infof("Persistent data volume %v attatched to master", masterDiskId)
	conn.cluster.Status.Cloud.AWS.VolumeId = masterDiskId
	time.Sleep(15 * time.Second)
	r2, err := conn.ec2.CreateRoute(&_ec2.CreateRouteInput{
		RouteTableId:         StringP(conn.cluster.Status.Cloud.AWS.RouteTableId),
		DestinationCidrBlock: StringP(conn.cluster.Spec.Networking.MasterSubnet),
		InstanceId:           StringP(masterInstanceID),
	})
	Logger(conn.ctx).Debug("Created route to master", r2, err)
	if err != nil {
		return &node, err
	}
	Logger(conn.ctx).Infof("Master route to route table %v for ip %v created", conn.cluster.Status.Cloud.AWS.RouteTableId, masterInstanceID)
	return &node, nil
}

func (conn *cloudConnector) waitForInstanceState(instanceId, state string) error {
	for {
		r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
			InstanceIds: []*string{StringP(instanceId)},
		})
		if err != nil {
			return err
		}
		curState := *r1.Reservations[0].Instances[0].State.Name
		if curState == state {
			break
		}
		Logger(conn.ctx).Infof("Waiting for instance %v to be %v (currently %v)", instanceId, state, curState)
		Logger(conn.ctx).Infof("Sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (conn *cloudConnector) ensurePd(name, diskType string, sizeGb int64) (string, error) {
	volumeId, err := conn.findPD(name)
	if err != nil {
		return volumeId, err
	}
	if volumeId == "" {
		// name := cluster.Spec.ctx.KubernetesMasterName + "-pd"
		r1, err := conn.ec2.CreateVolume(&_ec2.CreateVolumeInput{
			AvailabilityZone: &conn.cluster.Spec.Cloud.Zone,
			VolumeType:       &diskType,
			Size:             Int64P(sizeGb),
		})
		Logger(conn.ctx).Debug("Created master pd", r1, err)
		if err != nil {
			return "", err
		}
		volumeId = *r1.VolumeId
		Logger(conn.ctx).Infof("Master disk with size %vGB, type %v created", sizeGb, conn.cluster.Spec.MasterDiskType)

		time.Sleep(preTagDelay)
		err = conn.addTag(volumeId, "Name", name)
		if err != nil {
			return volumeId, err
		}
		err = conn.addTag(volumeId, "KubernetesCluster", conn.cluster.Name)
		if err != nil {
			return volumeId, err
		}
	}
	return volumeId, nil
}

func (conn *cloudConnector) findPD(name string) (string, error) {
	// name := cluster.Spec.ctx.KubernetesMasterName + "-pd"
	Logger(conn.ctx).Infof("Searching master pd %v", name)
	r1, err := conn.ec2.DescribeVolumes(&_ec2.DescribeVolumesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("availability-zone"),
				Values: []*string{
					StringP(conn.cluster.Spec.Cloud.Zone),
				},
			},
			{
				Name: StringP("tag:Name"),
				Values: []*string{
					StringP(name),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Retrieved master pd", r1, err)
	if err != nil {
		return "", err
	}
	if len(r1.Volumes) > 0 {
		Logger(conn.ctx).Infof("Found master pd %v", name)
		return *r1.Volumes[0].VolumeId, nil
	}
	Logger(conn.ctx).Infof("Master pd %v not found", name)
	return "", nil
}

func (conn *cloudConnector) createReserveIP(masterNG *api.NodeGroup) (string, error) {
	// Check that MASTER_RESERVED_IP looks like an IPv4 address
	// if match, _ := regexp.MatchString("^[0-9]+.[0-9]+.[0-9]+.[0-9]+$", cluster.Spec.ctx.MasterReservedIP); !match {
	r1, err := conn.ec2.AllocateAddress(&_ec2.AllocateAddressInput{
		Domain: StringP("vpc"),
	})
	Logger(conn.ctx).Debug("Allocated elastic IP", r1, err)
	if err != nil {
		return "", err
	}
	time.Sleep(5 * time.Second)

	Logger(conn.ctx).Infof("Elastic IP %v allocated", conn.cluster.Spec.MasterReservedIP)

	return *r1.PublicIp, nil
}

func (conn *cloudConnector) createMasterInstance(name string, ng *api.NodeGroup) (string, error) {
	script, err := conn.renderStartupScript(ng, "")
	if err != nil {
		return "", err
	}

	req := &_ec2.RunInstancesInput{
		ImageId:  StringP(conn.cluster.Spec.Cloud.InstanceImage),
		MaxCount: Int64P(1),
		MinCount: Int64P(1),
		//// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html
		BlockDeviceMappings: []*_ec2.BlockDeviceMapping{
			// MASTER_BLOCK_DEVICE_MAPPINGS
			{
				// https://github.com/appscode/kubernetes/blob/55d9dec8eb5eb02e1301045b7b81bbac689c86a1/cluster/aws/util.sh#L397
				DeviceName: StringP(conn.cluster.Status.Cloud.AWS.RootDeviceName),
				Ebs: &_ec2.EbsBlockDevice{
					DeleteOnTermination: TrueP(),
					VolumeSize:          Int64P(ng.Spec.Template.Spec.DiskSize),
					VolumeType:          StringP(ng.Spec.Template.Spec.DiskType),
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
		IamInstanceProfile: &_ec2.IamInstanceProfileSpecification{
			Name: StringP(conn.cluster.Spec.Cloud.AWS.IAMProfileMaster),
		},
		InstanceType: StringP(ng.Spec.Template.Spec.SKU),
		KeyName:      StringP(conn.cluster.Spec.Cloud.SSHKeyName),
		Monitoring: &_ec2.RunInstancesMonitoringEnabled{
			Enabled: TrueP(),
		},
		NetworkInterfaces: []*_ec2.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: TrueP(),
				DeleteOnTermination:      TrueP(),
				DeviceIndex:              Int64P(0),
				Groups: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.MasterSGId),
				},
				PrivateIpAddresses: []*_ec2.PrivateIpAddressSpecification{
					{
						PrivateIpAddress: StringP(conn.cluster.Spec.MasterInternalIP),
						Primary:          TrueP(),
					},
				},
				SubnetId: StringP(conn.cluster.Status.Cloud.AWS.SubnetId),
			},
		},
		UserData: StringP(base64.StdEncoding.EncodeToString([]byte(script))),
	}
	fmt.Println(req)
	r1, err := conn.ec2.RunInstances(req)
	Logger(conn.ctx).Debug("Created instance", r1, err)
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("Instance %v created with role %v", name, api.RoleMaster)
	instanceID := *r1.Instances[0].InstanceId
	time.Sleep(preTagDelay)

	err = conn.addTag(instanceID, "Name", conn.namer.MasterName())
	if err != nil {
		return instanceID, err
	}
	err = conn.addTag(instanceID, "Role", api.RoleMaster)
	if err != nil {
		return "", err
	}
	err = conn.addTag(instanceID, "KubernetesCluster", conn.cluster.Name)
	if err != nil {
		return "", err
	}
	return instanceID, nil
}

func (conn *cloudConnector) getInstancePublicIP(instanceID string) (string, bool, error) {
	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		InstanceIds: []*string{StringP(instanceID)},
	})
	Logger(conn.ctx).Debug("Retrieved Public IP for Instance", r1, err)
	if err != nil {
		return "", false, err
	}
	if r1.Reservations != nil && r1.Reservations[0].Instances != nil && r1.Reservations[0].Instances[0].NetworkInterfaces != nil {
		Logger(conn.ctx).Infof("Public ip for instance id %v retrieved", instanceID)
		return *r1.Reservations[0].Instances[0].NetworkInterfaces[0].Association.PublicIp, true, nil
	}
	return "", false, nil
}

func (conn *cloudConnector) listInstances(groupName string) ([]*api.NodeInfo, error) {
	r2, err := conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			StringP(groupName),
		},
	})
	if err != nil {
		return nil, err
	}
	instances := make([]*api.NodeInfo, 0)
	for _, group := range r2.AutoScalingGroups {
		for _, instance := range group.Instances {
			ki, err := conn.newKubeInstance(*instance.InstanceId)
			if err != nil {
				return nil, err
			}
			instances = append(instances, ki)
		}
	}
	return instances, nil
}

func (conn *cloudConnector) newKubeInstance(instanceID string) (*api.NodeInfo, error) {
	var err error
	var instance *_ec2.DescribeInstancesOutput
	attempt := 0
	wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		instance, err = conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
			InstanceIds: []*string{StringP(instanceID)},
		})
		fmt.Println(*instance.Reservations[0].Instances[0].State.Name, instanceID)
		Logger(conn.ctx).Infof("Attempt %v: Describing instance ...", attempt)
		if *instance.Reservations[0].Instances[0].State.Name != "pending" {
			return true, nil
		}
		return false, err
	})

	Logger(conn.ctx).Debug("Retrieved instance ", instance, err)
	if err != nil {
		return nil, ErrNotFound
	}

	if *instance.Reservations[0].Instances[0].State.Name != "running" {
		return &api.NodeInfo{}, nil
	}

	// Don't reassign internal_ip for AWS to keep the fixed 172.20.0.9 for master_internal_ip
	i := api.NodeInfo{
		Name:       *instance.Reservations[0].Instances[0].PrivateDnsName,
		ExternalID: instanceID,
		PublicIP:   *instance.Reservations[0].Instances[0].PublicIpAddress,
		PrivateIP:  *instance.Reservations[0].Instances[0].PrivateIpAddress,
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

	return &i, nil
}

func (conn *cloudConnector) allocateElasticIp() (string, error) {
	r1, err := conn.ec2.AllocateAddress(&_ec2.AllocateAddressInput{
		Domain: StringP("vpc"),
	})
	Logger(conn.ctx).Debug("Allocated elastic IP", r1, err)
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("Elastic IP %v allocated", *r1.PublicIp)
	time.Sleep(5 * time.Second)
	return *r1.PublicIp, nil
}

func (conn *cloudConnector) assignIPToInstance(reservedIP, instanceID string) error {
	r1, err := conn.ec2.DescribeAddresses(&_ec2.DescribeAddressesInput{
		PublicIps: []*string{StringP(reservedIP)},
	})
	Logger(conn.ctx).Debug("Retrieved allocation ID for elastic IP", r1, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Found allocation id %v for elastic IP %v", r1.Addresses[0].AllocationId, reservedIP)
	time.Sleep(1 * time.Minute)

	r2, err := conn.ec2.AssociateAddress(&_ec2.AssociateAddressInput{
		InstanceId:   StringP(instanceID),
		AllocationId: r1.Addresses[0].AllocationId,
	})
	Logger(conn.ctx).Debug("Attached IP to instance", r2, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("IP %v attached to instance %v", reservedIP, instanceID)
	return nil
}

func (conn *cloudConnector) createLaunchConfiguration(name, token string, ng *api.NodeGroup) error {
	script, err := conn.renderStartupScript(ng, token)
	if err != nil {
		return err
	}
	configuration := &autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  StringP(name),
		AssociatePublicIpAddress: TrueP(), // BoolP(conn.cluster.Spec.EnableNodePublicIP),
		// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/block-device-mapping-concepts.html
		BlockDeviceMappings: []*autoscaling.BlockDeviceMapping{
			// NODE_BLOCK_DEVICE_MAPPINGS
			{
				// https://github.com/appscode/kubernetes/blob/55d9dec8eb5eb02e1301045b7b81bbac689c86a1/cluster/aws/util.sh#L397
				DeviceName: StringP(conn.cluster.Status.Cloud.AWS.RootDeviceName),
				Ebs: &autoscaling.Ebs{
					DeleteOnTermination: TrueP(),
					VolumeSize:          Int64P(ng.Spec.Template.Spec.DiskSize),
					VolumeType:          StringP(ng.Spec.Template.Spec.DiskType),
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
		IamInstanceProfile: StringP(conn.cluster.Spec.Cloud.AWS.IAMProfileNode),
		ImageId:            StringP(conn.cluster.Spec.Cloud.InstanceImage),
		InstanceType:       StringP(ng.Spec.Template.Spec.SKU),
		KeyName:            StringP(conn.cluster.Spec.Cloud.SSHKeyName),
		SecurityGroups: []*string{
			StringP(conn.cluster.Status.Cloud.AWS.NodeSGId),
		},
		UserData: StringP(base64.StdEncoding.EncodeToString([]byte(script))),
	}
	if ng.Spec.Template.Spec.Type == api.NodeTypeSpot {
		configuration.SpotPrice = StringP(strconv.FormatFloat(ng.Spec.Template.Spec.SpotPriceMax, 'f', -1, 64))
	}
	r1, err := conn.autoscale.CreateLaunchConfiguration(configuration)
	Logger(conn.ctx).Debug("Created node configuration", r1, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Info("Node configuration created assuming node public ip is enabled")
	return nil
}

func (conn *cloudConnector) createAutoScalingGroup(name, launchConfig string, count int64) error {
	r2, err := conn.autoscale.CreateAutoScalingGroup(&autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: StringP(name),
		MaxSize:              Int64P(count),
		MinSize:              Int64P(count),
		DesiredCapacity:      Int64P(count),
		AvailabilityZones: []*string{
			StringP(conn.cluster.Spec.Cloud.Zone),
		},
		LaunchConfigurationName: StringP(launchConfig),
		Tags: []*autoscaling.Tag{
			{
				Key:          StringP("Name"),
				ResourceId:   StringP(name),
				ResourceType: StringP("auto-scaling-group"),
				Value:        StringP(name), // node instance prefix LN_1042
			},
			{
				Key:          StringP("Role"),
				ResourceId:   StringP(name),
				ResourceType: StringP("auto-scaling-group"),
				Value:        StringP(conn.cluster.Name + "-node"),
			},
			{
				Key:          StringP("KubernetesCluster"),
				ResourceId:   StringP(name),
				ResourceType: StringP("auto-scaling-group"),
				Value:        StringP(conn.cluster.Name),
			},
		},
		VPCZoneIdentifier: StringP(conn.cluster.Status.Cloud.AWS.SubnetId),
	})
	Logger(conn.ctx).Debug("Created autoscaling group", r2, err)
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Autoscaling group %v created", name)
	return nil
}

func (conn *cloudConnector) detectMaster() error {
	masterID, err := conn.getInstanceIDFromName(conn.namer.MasterName())
	if masterID == "" {
		Logger(conn.ctx).Info("Could not detect Kubernetes master node.  Make sure you've launched a cluster with appctl.")
		//os.Exit(0)
	}
	if err != nil {
		return err
	}

	masterIP, _, err := conn.getInstancePublicIP(masterID)
	if masterIP == "" {
		Logger(conn.ctx).Info("Could not detect Kubernetes master node IP.  Make sure you've launched a cluster with appctl")
		os.Exit(0)
	}
	Logger(conn.ctx).Infof("Using master: %v (external IP: %v)", conn.namer.MasterName(), masterIP)
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) getInstanceIDFromName(tagName string) (string, error) {
	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("tag:Name"),
				Values: []*string{
					StringP(tagName),
				},
			},
			{
				Name: StringP("instance-state-name"),
				Values: []*string{
					StringP("running"),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Retrieved instace via name", r1, err)
	if err != nil {
		return "", err
	}
	if r1.Reservations != nil && r1.Reservations[0].Instances != nil {
		return *r1.Reservations[0].Instances[0].InstanceId, nil
	}
	return "", nil
}

func (conn *cloudConnector) releaseReservedIP(publicIP string) error {
	r1, err := conn.ec2.DescribeAddresses(&_ec2.DescribeAddressesInput{
		PublicIps: []*string{
			StringP(publicIP),
		},
	})
	if err != nil {
		return err
	}

	_, err = conn.ec2.ReleaseAddress(&_ec2.ReleaseAddressInput{
		AllocationId: r1.Addresses[0].AllocationId,
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Elastic IP for cluster %v is deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteSecurityGroup() error {
	r, err := conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.VpcId),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	for _, sg := range r.SecurityGroups {
		if len(sg.IpPermissions) > 0 {
			_, err := conn.ec2.RevokeSecurityGroupIngress(&_ec2.RevokeSecurityGroupIngressInput{
				GroupId:       sg.GroupId,
				IpPermissions: sg.IpPermissions,
			})
			if err != nil {
				return err
			}
		}

		if len(sg.IpPermissionsEgress) > 0 {
			_, err := conn.ec2.RevokeSecurityGroupEgress(&_ec2.RevokeSecurityGroupEgressInput{
				GroupId:       sg.GroupId,
				IpPermissions: sg.IpPermissionsEgress,
			})
			if err != nil {
				return err
			}
		}
	}

	for _, sg := range r.SecurityGroups {
		_, err := conn.ec2.DeleteSecurityGroup(&_ec2.DeleteSecurityGroupInput{
			GroupId: sg.GroupId,
		})
		if err != nil {
			return err
		}
	}
	Logger(conn.ctx).Infof("Security groups for cluster %v is deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteSubnetId() error {
	r, err := conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.VpcId),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	for _, subnet := range r.Subnets {
		_, err := conn.ec2.DeleteSubnet(&_ec2.DeleteSubnetInput{
			SubnetId: subnet.SubnetId,
		})
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Subnet ID in VPC %v is deleted", *subnet.SubnetId)
	}
	return nil
}

func (conn *cloudConnector) deleteInternetGateway() error {
	r1, err := conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("attachment.vpc-id"),
				Values: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.VpcId),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	for _, igw := range r1.InternetGateways {
		_, err := conn.ec2.DetachInternetGateway(&_ec2.DetachInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
			VpcId:             StringP(conn.cluster.Status.Cloud.AWS.VpcId),
		})
		if err != nil {
			return err
		}

		_, err = conn.ec2.DeleteInternetGateway(&_ec2.DeleteInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
		})
		if err != nil {
			return err
		}
	}
	Logger(conn.ctx).Infof("Internet gateway for cluster %v are deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteRouteTable() error {
	r1, err := conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(conn.cluster.Status.Cloud.AWS.VpcId),
				},
			},
		},
	})

	if err != nil {
		fmt.Println(err)
		return err
	}
	for _, rt := range r1.RouteTables {
		mainTable := false
		for _, assoc := range rt.Associations {
			if _aws.BoolValue(assoc.Main) {
				mainTable = true
			} else {
				_, err := conn.ec2.DisassociateRouteTable(&_ec2.DisassociateRouteTableInput{
					AssociationId: assoc.RouteTableAssociationId,
				})
				if err != nil {
					fmt.Println(err)
					return err
				}
			}
		}
		if !mainTable {
			_, err := conn.ec2.DeleteRouteTable(&_ec2.DeleteRouteTableInput{
				RouteTableId: rt.RouteTableId,
			})
			if err != nil {
				fmt.Println(err)
				return err
			}
		}
	}
	Logger(conn.ctx).Infof("Route tables for cluster %v are deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteDHCPOption() error {
	_, err := conn.ec2.AssociateDhcpOptions(&_ec2.AssociateDhcpOptionsInput{
		VpcId:         StringP(conn.cluster.Status.Cloud.AWS.VpcId),
		DhcpOptionsId: StringP("default"),
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	r, err := conn.ec2.DescribeDhcpOptions(&_ec2.DescribeDhcpOptionsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	for _, dhcp := range r.DhcpOptions {
		_, err = conn.ec2.DeleteDhcpOptions(&_ec2.DeleteDhcpOptionsInput{
			DhcpOptionsId: dhcp.DhcpOptionsId,
		})
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	Logger(conn.ctx).Infof("DHCP options for cluster %v are deleted", conn.cluster.Name)
	return err
}

func (conn *cloudConnector) deleteVpc() error {
	_, err := conn.ec2.DeleteVpc(&_ec2.DeleteVpcInput{
		VpcId: StringP(conn.cluster.Status.Cloud.AWS.VpcId),
	})

	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("VPC for cluster %v is deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteVolume() error {
	_, err := conn.ec2.DeleteVolume(&_ec2.DeleteVolumeInput{
		VolumeId: StringP(conn.cluster.Status.Cloud.AWS.VolumeId),
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("Master instance volume for cluster %v is deleted", conn.cluster.Spec.MasterDiskId)
	return nil
}

func (conn *cloudConnector) deleteSSHKey() error {
	var err error
	_, err = conn.ec2.DeleteKeyPair(&_ec2.DeleteKeyPairInput{
		KeyName: StringP(conn.cluster.Spec.Cloud.SSHKeyName),
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("SSH key for cluster %v is deleted", conn.cluster.Spec.MasterDiskId)
	//updates := &storage.SSHKey{IsDeleted: 1}
	//cond := &storage.SSHKey{PHID: cluster.Spec.ctx.SSHKeyPHID}
	// _, err = cluster.Spec.Store(ctx).Engine.Update(updates, cond)

	return err
}

func (conn *cloudConnector) deleteNetworkInterface(vpcId string) error {
	r, err := conn.ec2.DescribeNetworkInterfaces(&_ec2.DescribeNetworkInterfacesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(vpcId),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	for _, iface := range r.NetworkInterfaces {
		_, err = conn.ec2.DetachNetworkInterface(&_ec2.DetachNetworkInterfaceInput{
			AttachmentId: iface.Attachment.AttachmentId,
			Force:        TrueP(),
		})
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Minute)
		_, err = conn.ec2.DeleteNetworkInterface(&_ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: iface.NetworkInterfaceId,
		})
		if err != nil {
			return err
		}
	}
	Logger(conn.ctx).Infof("Network interfaces for cluster %v are deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteAutoScalingGroup(name string) error {
	_, err := conn.autoscale.DeleteAutoScalingGroup(&autoscaling.DeleteAutoScalingGroupInput{
		ForceDelete:          TrueP(),
		AutoScalingGroupName: StringP(name),
	})
	Logger(conn.ctx).Infof("Auto scaling group %v is deleted for cluster %v", name, conn.cluster.Name)
	return err
}

func (conn *cloudConnector) deleteLaunchConfiguration(name string) error {
	_, err := conn.autoscale.DeleteLaunchConfiguration(&autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: StringP(name),
	})
	Logger(conn.ctx).Infof("Launch configuration %v os deleted for cluster %v", name, conn.cluster.Name)
	return err
}

func (conn *cloudConnector) deleteMaster() error {
	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("tag:Role"),
				Values: []*string{
					StringP(api.RoleMaster),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	masterInstances := make([]*string, 0)
	for _, reservation := range r1.Reservations {
		for _, instance := range reservation.Instances {
			masterInstances = append(masterInstances, instance.InstanceId)
		}
	}
	fmt.Printf("TerminateInstances %v", stringutil.Join(masterInstances, ","))
	Logger(conn.ctx).Infof("Terminating master instance for cluster %v", conn.cluster.Name)
	_, err = conn.ec2.TerminateInstances(&_ec2.TerminateInstancesInput{
		InstanceIds: masterInstances,
	})
	if err != nil {
		return err
	}
	instanceInput := &_ec2.DescribeInstancesInput{
		InstanceIds: masterInstances,
	}
	err = conn.ec2.WaitUntilInstanceTerminated(instanceInput)
	Logger(conn.ctx).Infof("Master instance for cluster %v is terminated", conn.cluster.Name)
	return err
}

func (conn *cloudConnector) deleteGroupInstances(ng *api.NodeGroup, instance string) error {
	if _, err := conn.autoscale.DetachInstances(&autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           StringP(ng.Name),
		InstanceIds:                    StringPSlice([]string{instance}),
		ShouldDecrementDesiredCapacity: BoolP(true),
	}); err != nil {
		return err
	}

	Logger(conn.ctx).Infof("Terminating instance for cluster %v", conn.cluster.Name)
	if _, err := conn.ec2.TerminateInstances(&_ec2.TerminateInstancesInput{
		InstanceIds: StringPSlice([]string{instance}),
	}); err != nil {
		return err
	}
	instanceInput := &_ec2.DescribeInstancesInput{
		InstanceIds: StringPSlice([]string{instance}),
	}
	return conn.ec2.WaitUntilInstanceTerminated(instanceInput)
}

func (conn *cloudConnector) ensureInstancesDeleted() error {
	const desiredState = "terminated"

	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return err
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
				ris = append(ris, StringP(instance))
			}
		}
		fmt.Println("Waiting for instances to terminate", stringutil.Join(ris, ","))

		r2, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{InstanceIds: ris})
		if err != nil {
			return err
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

func (conn *cloudConnector) describeGroupInfo(instanceGroup string) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	groups := make([]*string, 0)
	groups = append(groups, StringP(instanceGroup))
	r1, err := conn.autoscale.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: groups,
	})
	if err != nil {
		return nil, err
	}
	return r1, nil
}

func (conn *cloudConnector) getInstancePublicDNS(providerID string) (string, error) {
	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("instance-id"),
				Values: []*string{
					StringP(providerID),
				},
			},
			{
				Name: StringP("tag:KubernetesCluster"),
				Values: []*string{
					StringP(conn.cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return "", err
	}
	return *r1.Reservations[0].Instances[0].PublicDnsName, nil
}

func (conn *cloudConnector) getExistingLaunchConfigurationTemplate(ng *api.NodeGroup) (string, error) {
	as, err := conn.describeGroupInfo(ng.Name)
	if err != nil {
		return "", err
	}
	return *as.AutoScalingGroups[0].LaunchConfigurationName, nil
}

//Launch configuration template update
func (conn *cloudConnector) updateLaunchConfigurationTemplate(ng *api.NodeGroup, token string) error {
	newConfigurationTemplate := conn.namer.LaunchConfigName(ng.Spec.Template.Spec.SKU)

	if err := conn.createLaunchConfiguration(newConfigurationTemplate, token, ng); err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	oldConfigurationTemplate, err := conn.getExistingLaunchConfigurationTemplate(ng)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}

	fmt.Println("Updating autoscalling group")
	_, err = conn.autoscale.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName:    StringP(ng.Name),
		LaunchConfigurationName: StringP(newConfigurationTemplate),
	})
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}

	err = conn.deleteLaunchConfiguration(oldConfigurationTemplate)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	return nil
}

// ref: https://github.com/kubernetes/kubernetes/blob/0b9efaeb34a2fc51ff8e4d34ad9bc6375459c4a4/pkg/cloudprovider/providers/aws/instances.go#L43

// providerId represents the id for an instance in the kubernetes API;
// the following form
//  * aws:///<zone>/<awsInstanceId>
//  * aws:////<awsInstanceId>
//  * <awsInstanceId>

// splitProviderID extracts the awsInstanceID from the kubernetesInstanceID
func splitProviderID(providerId string) (string, error) {
	if !strings.HasPrefix(providerId, "aws://") {
		// Assume a bare aws volume id (vol-1234...)
		// Build a URL with an empty host (AZ)
		providerId = "aws://" + "/" + "/" + providerId
	}
	url, err := url.Parse(providerId)
	if err != nil {
		return "", errors.Errorf("invalid instance name (%s): %v", providerId, err)
	}
	if url.Scheme != "aws" {
		return "", errors.Errorf("invalid scheme for AWS instance (%s)", providerId)
	}

	awsID := ""
	tokens := strings.Split(strings.Trim(url.Path, "/"), "/")
	if len(tokens) == 1 {
		// instanceId
		awsID = tokens[0]
	} else if len(tokens) == 2 {
		// az/instanceId
		awsID = tokens[1]
	}

	// We sanity check the resulting volume; the two known formats are
	// i-12345678 and i-12345678abcdef01
	// TODO: Regex match?
	if awsID == "" || strings.Contains(awsID, "/") || !strings.HasPrefix(awsID, "i-") {
		return "", errors.Errorf("Invalid format for AWS instance (%s)", providerId)
	}

	return awsID, nil
}
