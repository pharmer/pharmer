/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package aws

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	clusterapi_aws "pharmer.dev/pharmer/apis/v1alpha1/aws"
	"pharmer.dev/pharmer/cloud"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	"github.com/appscode/go/wait"
	"github.com/aws/aws-sdk-go/aws"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	_elb "github.com/aws/aws-sdk-go/service/elb"
	_iam "github.com/aws/aws-sdk-go/service/iam"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	awssts "github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	bastion = "bastion"
)

type cloudConnector struct {
	*cloud.Scope

	namer     namer
	ec2       *_ec2.EC2
	elb       *_elb.ELB
	iam       *_iam.IAM
	autoscale *autoscaling.AutoScaling
	s3        *_s3.S3
}

func newconnector(cm *ClusterManager) (*cloudConnector, error) {
	cred, err := cm.StoreProvider.Credentials().Get(cm.Cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.AWS{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cm.Cluster.Spec.Config.CredentialName, err)
	}

	config := &_aws.Config{
		Region:      &cm.Cluster.Spec.Config.Cloud.Region,
		Credentials: credentials.NewStaticCredentials(typed.AccessKeyID(), typed.SecretAccessKey(), ""),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	conn := cloudConnector{
		Scope:     cm.Scope,
		namer:     namer{cm.Cluster},
		ec2:       _ec2.New(sess),
		elb:       _elb.New(sess),
		iam:       _iam.New(sess),
		autoscale: autoscaling.New(sess),
		s3:        _s3.New(sess),
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		return nil, errors.Errorf("credential %s does not have necessary authorization. Reason: %s", cm.Cluster.Spec.Config.CredentialName, msg)
	}
	return &conn, nil
}

func (cm *ClusterManager) SetCloudConnector() error {
	var err error
	if cm.conn, err = newconnector(cm); err != nil {
		return err
	}

	return nil
}

func (conn *cloudConnector) getInstanceRootDeviceSize(instance *ec2.Instance) (*int64, error) {
	for _, bdm := range instance.BlockDeviceMappings {
		if aws.StringValue(bdm.DeviceName) == aws.StringValue(instance.RootDeviceName) {
			input := &ec2.DescribeVolumesInput{
				VolumeIds: []*string{bdm.Ebs.VolumeId},
			}

			out, err := conn.ec2.DescribeVolumes(input)
			if err != nil {
				return nil, err
			}

			if len(out.Volumes) == 0 {
				return nil, errors.Errorf("no volumes found for id %q", aws.StringValue(bdm.Ebs.VolumeId))
			}

			return out.Volumes[0].Size, nil
		}
	}
	return nil, nil
}

// Returns true if unauthorized
func (conn *cloudConnector) IsUnauthorized() (bool, string) {
	policies := make(map[string]string)
	var marker *string
	for {
		resp, err := conn.iam.ListPolicies(&_iam.ListPoliciesInput{
			MaxItems: types.Int64P(1000),
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
	conn.Cluster.Spec.Config.Cloud.OS = "ubuntu"
	r1, err := conn.ec2.DescribeImages(&_ec2.DescribeImagesInput{
		Owners: []*string{types.StringP("099720109477")},
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("name"),
				Values: []*string{
					types.StringP("ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-20170619.1"),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	conn.Cluster.Spec.Config.Cloud.InstanceImage = *r1.Images[0].ImageId
	log.Infof("Ubuntu image with %v detected", conn.Cluster.Spec.Config.Cloud.InstanceImage)
	return nil
}

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	return nil
}

func (conn *cloudConnector) getIAMProfile() (bool, error) {
	r1, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileMaster})
	if r1.InstanceProfile == nil {
		return false, err
	}
	r2, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileNode})
	if r2.InstanceProfile == nil {
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) ensureIAMProfile() error {
	r1, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileMaster})
	if err != nil && !strings.Contains(err.Error(), "NoSuchEntity") {
		return err
	}
	if r1.InstanceProfile == nil {
		err := conn.createIAMProfile(api.RoleMaster, conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileMaster)
		if err != nil {
			return err
		}
		log.Infof("Master instance profile %v created", conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileMaster)
	}
	r2, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileNode})
	if err != nil && !strings.Contains(err.Error(), "NoSuchEntity") {
		return err
	}
	if r2.InstanceProfile == nil {
		err := conn.createIAMProfile(api.RoleNode, conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileNode)
		if err != nil {
			return err
		}
		log.Infof("Node instance profile %v created", conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileNode)
	}
	return nil
}

func (conn *cloudConnector) deleteIAMProfile() {
	conn.deleteRolePolicy(conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileMaster)
	conn.deleteRolePolicy(conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileNode)
}

func (conn *cloudConnector) deleteRolePolicy(role string) {
	if _, err := conn.iam.RemoveRoleFromInstanceProfile(&_iam.RemoveRoleFromInstanceProfileInput{
		InstanceProfileName: &role,
		RoleName:            &role,
	}); err != nil {
		log.Infoln("Failed to remove role from instance profile", role, err)
	}

	if _, err := conn.iam.DeleteRolePolicy(&_iam.DeleteRolePolicyInput{
		PolicyName: &role,
		RoleName:   &role,
	}); err != nil {
		log.Infoln("Failed to delete role policy", role, err)
	}

	if role == conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileMaster {
		if _, err := conn.iam.DeleteRolePolicy(&_iam.DeleteRolePolicyInput{
			PolicyName: &conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileNode,
			RoleName:   &role,
		}); err != nil {
			log.Infoln("Failed to delete role policy", role, err)
		}
		if _, err := conn.iam.DeleteRolePolicy(&_iam.DeleteRolePolicyInput{
			PolicyName: types.StringP(conn.namer.ControlPlanePolicyName()),
			RoleName:   &role,
		}); err != nil {
			log.Infoln("Failed to delete role policy", role, err)
		}
	}

	if _, err := conn.iam.DeleteRole(&_iam.DeleteRoleInput{
		RoleName: &role,
	}); err != nil {
		log.Infoln("Failed to delete role", role, err)
	}

	if _, err := conn.iam.DeleteInstanceProfile(&_iam.DeleteInstanceProfileInput{
		InstanceProfileName: &role,
	}); err != nil {
		log.Infoln("Failed to delete instance profile", role, err)
	}
}

func (conn *cloudConnector) createIAMProfile(role, key string) error {
	// ref: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/e6bb385084bc4bc067819391d2179b55d8b96e17/cmd/clusterawsadm/cmd/alpha/bootstrap/bootstrap.go#L131
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		fmt.Printf("Error: %v", err)
		return nil
	}

	accountID, err := awssts.New(sess).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		log.Debugf("Error getting account ID")
		return err
	}

	reqRole := &_iam.CreateRoleInput{RoleName: &key}
	if role == api.RoleMaster {
		reqRole.AssumeRolePolicyDocument = types.StringP(strings.TrimSpace(IAMMasterRole))
	} else {
		reqRole.AssumeRolePolicyDocument = types.StringP(strings.TrimSpace(IAMNodeRole))
	}
	r1, err := conn.iam.CreateRole(reqRole)
	log.Debug("Created IAM role", r1, err)
	log.Infof("IAM role %v created", key)
	if err != nil {
		return err
	}

	if role == api.RoleMaster {
		controlPlanePolicy := &_iam.PutRolePolicyInput{
			RoleName:       &key,
			PolicyName:     &key,
			PolicyDocument: types.StringP(strings.TrimSpace(IAMMasterPolicy)),
		}
		if _, err := conn.iam.PutRolePolicy(controlPlanePolicy); err != nil {
			log.Debugf("Error attaching control-plane policy to control-plane instance profile")
			return err
		}

		controllerPolicy := &_iam.PutRolePolicyInput{
			RoleName:       &key,
			PolicyName:     types.StringP(strings.Replace(key, "master", "controller", 1)),
			PolicyDocument: types.StringP(strings.TrimSpace(strings.Replace(IAMControllerPolicy, "ACCOUNT_ID", *accountID.Account, -1))),
		}
		if _, err := conn.iam.PutRolePolicy(controllerPolicy); err != nil {
			log.Debugf("Error attaching controller policy to control-plane instance profile")
			return err
		}

		nodePolicy := &_iam.PutRolePolicyInput{
			RoleName:       &key,
			PolicyName:     types.StringP(strings.Replace(key, "master", "node", 1)),
			PolicyDocument: types.StringP(strings.TrimSpace(IAMNodePolicy)),
		}
		if _, err := conn.iam.PutRolePolicy(nodePolicy); err != nil {
			log.Debugf("Error attaching node policy to control-plane instance profile")
			return err
		}
	} else {
		nodePolicy := &_iam.PutRolePolicyInput{
			RoleName:       &key,
			PolicyName:     &key,
			PolicyDocument: types.StringP(strings.TrimSpace(IAMNodePolicy)),
		}
		if _, err := conn.iam.PutRolePolicy(nodePolicy); err != nil {
			log.Debugf("Error attaching node policy to node instance profile")
			return err
		}
	}

	r3, err := conn.iam.CreateInstanceProfile(&_iam.CreateInstanceProfileInput{
		InstanceProfileName: &key,
	})
	log.Debug("Created IAM instance-policy", r3, err)
	if err != nil {
		return err
	}
	log.Infof("IAM instance-policy %v created", key)

	r4, err := conn.iam.AddRoleToInstanceProfile(&_iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: &key,
		RoleName:            &key,
	})
	log.Debug("Added IAM role to instance-policy", r4, err)
	if err != nil {
		return err
	}
	log.Infof("IAM role %v added to instance-policy %v", key, key)
	return nil
}

func (conn *cloudConnector) getPublicKey() (bool, error) {
	resp, err := conn.ec2.DescribeKeyPairs(&_ec2.DescribeKeyPairsInput{
		KeyNames: types.StringPSlice([]string{conn.Cluster.Spec.Config.Cloud.SSHKeyName}),
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
		KeyName:           types.StringP(conn.Cluster.Spec.Config.Cloud.SSHKeyName),
		PublicKeyMaterial: conn.Certs.SSHKey.PublicKey,
	})
	log.Debug("Imported SSH key", resp, err)
	// TODO ignore "InvalidKeyPair.Duplicate" error
	if err != nil {
		log.Info("Error importing public key", resp, err)
		//os.Exit(1)
		return err

	}

	return nil
}

func (conn *cloudConnector) getVpc() (string, bool, error) {
	log.Infof("Checking VPC tagged with %v", conn.Cluster.Name)
	r1, err := conn.ec2.DescribeVpcs(&_ec2.DescribeVpcsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:Name"),
				Values: []*string{
					types.StringP(conn.namer.VPCName()),
				},
			},
			{
				Name: types.StringP("tag-key"),
				Values: []*string{
					types.StringP("kubernetes.io/cluster/" + conn.Cluster.Name),
				},
			},
		},
	})
	log.Debug("VPC described", r1, err)
	if len(r1.Vpcs) > 0 {
		log.Infof("VPC %v found", *r1.Vpcs[0].VpcId)
		return *r1.Vpcs[0].VpcId, true, nil
	}

	return "", false, errors.New("VPC not found")
}

func (conn *cloudConnector) setupVpc() (string, error) {
	log.Info("No VPC found, creating new VPC")
	r2, err := conn.ec2.CreateVpc(&_ec2.CreateVpcInput{
		CidrBlock: types.StringP(conn.Cluster.Spec.Config.Cloud.AWS.VpcCIDR),
	})
	log.Debug("VPC created", r2, err)

	if err != nil {
		return "", err
	}
	log.Infof("VPC %v created", *r2.Vpc.VpcId)

	wReq := &ec2.DescribeVpcsInput{VpcIds: []*string{r2.Vpc.VpcId}}
	if err := conn.ec2.WaitUntilVpcAvailable(wReq); err != nil {
		return "", errors.Wrapf(err, "failed to wait for vpc %q", *r2.Vpc.VpcId)
	}

	tags := map[string]string{
		"Name": conn.Cluster.Name + "-vpc",
		"kubernetes.io/cluster/" + conn.Cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
	}

	vpcID := *r2.Vpc.VpcId
	if err := conn.addTags(vpcID, tags); err != nil {
		return "", err
	}

	return vpcID, nil
}

func (conn *cloudConnector) addTags(id string, tags map[string]string) error {
	for key, val := range tags {
		if err := conn.addTag(id, key, val); err != nil {
			return err
		}
	}
	return nil
}

func (conn *cloudConnector) addTag(id string, key string, value string) error {
	_, err := conn.ec2.CreateTags(&_ec2.CreateTagsInput{
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
	if err != nil {
		return errors.Wrapf(err, "failed to add tag %v:%v to id   %v", key, value, id)
	}
	log.Infof("Added tag %v:%v to id %v", key, value, id)
	return nil
}

func (conn *cloudConnector) getLoadBalancer() (string, error) {
	input := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: aws.StringSlice([]string{fmt.Sprintf("%s-apiserver", conn.Cluster.Name)}),
	}
	out, err := conn.elb.DescribeLoadBalancers(input)
	if err != nil {
		return "", err
	}
	if len(out.LoadBalancerDescriptions) == 0 {
		return "", nil
	}

	return *out.LoadBalancerDescriptions[0].DNSName, nil
}

func (conn *cloudConnector) deleteLoadBalancer() (bool, error) {
	log.Infof("deleting load balancer")
	input := &elb.DeleteLoadBalancerInput{
		LoadBalancerName: types.StringP(fmt.Sprintf("%s-apiserver", conn.Cluster.Name)),
	}
	if _, err := conn.elb.DeleteLoadBalancer(input); err != nil {
		return false, err
	}

	if err := wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		_, err := conn.getLoadBalancer()
		if strings.Contains(err.Error(), "LoadBalancerNotFound") {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return false, err
	}
	log.Infof("successfully deleted load balancer")
	return true, nil
}

func (conn *cloudConnector) setupLoadBalancer(publicSubnetID string) (string, error) {
	input := elb.CreateLoadBalancerInput{
		LoadBalancerName: types.StringP(fmt.Sprintf("%s-apiserver", conn.Cluster.Name)),
		Subnets:          []*string{types.StringP(publicSubnetID)},
		SecurityGroups:   []*string{types.StringP(conn.Cluster.Status.Cloud.AWS.MasterSGId)},
		Scheme:           types.StringP("Internet-facing"),
		Listeners: []*elb.Listener{
			{
				Protocol:         types.StringP("TCP"),
				LoadBalancerPort: types.Int64P(6443),
				InstanceProtocol: types.StringP("TCP"),
				InstancePort:     types.Int64P(6443),
			},
		},
		Tags: []*elb.Tag{
			{
				Key:   types.StringP("sigs.k8s.io/cluster-api-provider-aws/role"),
				Value: types.StringP("apiserver"),
			},
			{
				Key:   types.StringP(fmt.Sprintf("kubernetes.io/cluster/%s", conn.Cluster.Name)),
				Value: types.StringP("owned"),
			},
			{
				Key:   types.StringP("sigs.k8s.io/cluster-api-provider-aws/managed"),
				Value: types.StringP("true"),
			},
		},
	}

	output, err := conn.elb.CreateLoadBalancer(&input)
	if err != nil {
		return "", err
	}

	hc := &elb.ConfigureHealthCheckInput{
		LoadBalancerName: input.LoadBalancerName,
		HealthCheck: &elb.HealthCheck{
			Target:             types.StringP("TCP:6443"),
			Interval:           types.Int64P(int64((10 * time.Second).Seconds())),
			Timeout:            types.Int64P(int64((5 * time.Second).Seconds())),
			HealthyThreshold:   types.Int64P(5),
			UnhealthyThreshold: types.Int64P(3),
		},
	}

	if _, err := conn.elb.ConfigureHealthCheck(hc); err != nil {
		return "", err
	}

	return *output.DNSName, nil
}

func (conn *cloudConnector) getSubnet(vpcID, privacy string) (string, bool, error) {
	log.Infof("Checking for existing %s subnet", privacy)
	r1, err := conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("availabilityZone"),
				Values: []*string{
					types.StringP(conn.Cluster.Spec.Config.Cloud.Zone),
				},
			},
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(vpcID),
				},
			},
			{
				Name: types.StringP("tag:Name"),
				Values: []*string{
					types.StringP(fmt.Sprintf("%s-subnet-%s", conn.Cluster.Name, privacy)),
				},
			},
		},
	})
	log.Debug(fmt.Sprintf("Retrieved %s subnet", privacy), r1, err)
	if err != nil {
		return "", false, err
	}

	if len(r1.Subnets) == 0 {
		return "", false, errors.Errorf("No %s subnet found", privacy)
	}
	log.Infof("%s Subnet %v found with CIDR %v", privacy, r1.Subnets[0].SubnetId, r1.Subnets[0].CidrBlock)

	return *r1.Subnets[0].SubnetId, true, nil

}

func (conn *cloudConnector) setupPrivateSubnet(vpcID string) (string, error) {
	log.Info("No subnet found, creating new subnet")
	r2, err := conn.ec2.CreateSubnet(&_ec2.CreateSubnetInput{
		CidrBlock:        types.StringP(conn.Cluster.Spec.Config.Cloud.AWS.PrivateSubnetCIDR),
		VpcId:            types.StringP(vpcID),
		AvailabilityZone: types.StringP(conn.Cluster.Spec.Config.Cloud.Zone),
	})
	log.Debug("Created subnet", r2, err)
	if err != nil {
		return "", err
	}

	id := *r2.Subnet.SubnetId
	log.Infof("Subnet %v created", id)

	time.Sleep(preTagDelay)

	tags := map[string]string{
		"Name": conn.Cluster.Name + "-subnet-private",
		"kubernetes.io/cluster/" + conn.Cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    "common",
	}

	if err := conn.addTags(id, tags); err != nil {
		return "", err
	}

	return id, nil
}

func (conn *cloudConnector) setupPublicSubnet(vpcID string) (string, error) {
	log.Info("No subnet found, creating new subnet")
	r2, err := conn.ec2.CreateSubnet(&_ec2.CreateSubnetInput{
		CidrBlock:        types.StringP(conn.Cluster.Spec.Config.Cloud.AWS.PublicSubnetCIDR),
		VpcId:            types.StringP(vpcID),
		AvailabilityZone: types.StringP(conn.Cluster.Spec.Config.Cloud.Zone),
	})
	log.Debug("Created subnet", r2, err)
	if err != nil {
		return "", err
	}
	log.Infof("Subnet %v created", *r2.Subnet.SubnetId)

	wReq := &ec2.DescribeSubnetsInput{SubnetIds: []*string{r2.Subnet.SubnetId}}
	if err := conn.ec2.WaitUntilSubnetAvailable(wReq); err != nil {
		return "", errors.Wrapf(err, "failed to wait for subnet %q", *r2.Subnet.SubnetId)
	}

	attReq := &_ec2.ModifySubnetAttributeInput{
		MapPublicIpOnLaunch: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
		SubnetId: r2.Subnet.SubnetId,
	}
	if _, err := conn.ec2.ModifySubnetAttribute(attReq); err != nil {
		return "", err
	}

	id := *r2.Subnet.SubnetId

	tags := map[string]string{
		"Name": conn.Cluster.Name + "-subnet-public",
		"kubernetes.io/cluster/" + conn.Cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    bastion,
	}

	if err := conn.addTags(id, tags); err != nil {
		return "", err
	}

	return id, nil
}

func (conn *cloudConnector) getInternetGateway(vpcID string) (string, bool, error) {
	log.Infof("Checking IGW with attached VPCID %v", vpcID)
	r1, err := conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("attachment.vpc-id"),
				Values: []*string{
					types.StringP(vpcID),
				},
			},
		},
	})
	log.Debug("Retrieved IGW", r1, err)
	if err != nil {
		return "", false, err
	}
	if len(r1.InternetGateways) == 0 {
		return "", false, errors.Errorf("IGW not found")
	}
	log.Infof("IGW %v found", *r1.InternetGateways[0].InternetGatewayId)
	return *r1.InternetGateways[0].InternetGatewayId, true, nil
}

func (conn *cloudConnector) setupInternetGateway(vpcID string) (string, error) {
	log.Info("No IGW found, creating new IGW")
	r2, err := conn.ec2.CreateInternetGateway(&_ec2.CreateInternetGatewayInput{})
	log.Debug("Created IGW", r2, err)
	if err != nil {
		return "", err
	}
	time.Sleep(preTagDelay)
	log.Infof("IGW %v created", *r2.InternetGateway.InternetGatewayId)

	r3, err := conn.ec2.AttachInternetGateway(&_ec2.AttachInternetGatewayInput{
		InternetGatewayId: types.StringP(*r2.InternetGateway.InternetGatewayId),
		VpcId:             types.StringP(vpcID),
	})
	log.Debug("Attached IGW to VPC", r3, err)
	if err != nil {
		return "", err
	}

	id := *r2.InternetGateway.InternetGatewayId
	log.Infof("Attached IGW %v to VPCID %v", id, vpcID)

	tags := map[string]string{
		"Name": conn.Cluster.Name + "-igw",
		"kubernetes.io/cluster/" + conn.Cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    "common",
	}

	if err := conn.addTags(id, tags); err != nil {
		return "", err
	}

	return id, nil
}

func (conn *cloudConnector) getNatGateway(vpcID string) (string, bool, error) {
	log.Infof("Checking NAT with attached VPCID %v", vpcID)
	r1, err := conn.ec2.DescribeNatGateways(&_ec2.DescribeNatGatewaysInput{
		Filter: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(vpcID),
				},
			},
			{
				Name: types.StringP("state"),
				Values: []*string{
					types.StringP("available"),
					types.StringP("pending"),
				},
			},
		},
	})
	log.Debug("Retrieved NAT", r1, err)
	if err != nil {
		return "", false, err
	}
	if len(r1.NatGateways) == 0 {
		return "", false, errors.Errorf("NAT not found")
	}

	log.Infof("NAT %v found", *r1.NatGateways[0].NatGatewayId)
	return *r1.NatGateways[0].NatGatewayId, true, nil
}

func (conn *cloudConnector) setupNatGateway(publicSubnetID string) (string, error) {
	log.Info("No NAT Gateway found, creating new")

	id, _, err := conn.allocateElasticIP()
	if err != nil {
		return "", err
	}

	out, err := conn.ec2.CreateNatGateway(&_ec2.CreateNatGatewayInput{
		AllocationId: types.StringP(id),
		SubnetId:     types.StringP(publicSubnetID),
	})

	wReq := &ec2.DescribeNatGatewaysInput{NatGatewayIds: []*string{out.NatGateway.NatGatewayId}}
	if err := conn.ec2.WaitUntilNatGatewayAvailable(wReq); err != nil {
		return "", errors.Wrapf(err, "failed to wait for nat gateway %q in subnet %q", *out.NatGateway.NatGatewayId, publicSubnetID)
	}

	log.Debug("Created NAT", out, err)
	if err != nil {
		return "", err
	}

	log.Infof("Nat Gateway %v created", *out.NatGateway.NatGatewayId)

	tags := map[string]string{
		"Name": conn.Cluster.Name + "-nat",
		"kubernetes.io/cluster/" + conn.Cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    "common",
	}

	if err := conn.addTags(*out.NatGateway.NatGatewayId, tags); err != nil {
		return "", err
	}

	return *out.NatGateway.NatGatewayId, nil
}

func (conn *cloudConnector) getRouteTable(privacy, vpcID string) (string, bool, error) {
	log.Infof("Checking %v route table for VPCID %v", privacy, vpcID)
	r1, err := conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(vpcID),
				},
			},
			{
				Name: types.StringP("tag-key"),
				Values: []*string{
					types.StringP(fmt.Sprintf("kubernetes.io/cluster/%s", conn.Cluster.Name)),
				},
			},
			{
				Name: types.StringP("tag:Name"),
				Values: []*string{
					types.StringP(fmt.Sprintf("%s-rt-%s", conn.Cluster.Name, privacy)),
				},
			},
		},
	})
	log.Infof("found %v route table", privacy)
	if err != nil {
		return "", false, err
	}
	if len(r1.RouteTables) == 0 {
		return "", false, errors.Errorf("Route table not found")
	}
	log.Infof("%v Route table %v found", privacy, *r1.RouteTables[0].RouteTableId)

	return *r1.RouteTables[0].RouteTableId, true, nil
}

func (conn *cloudConnector) setupRouteTable(privacy, vpcID, igwID, natID, publicSubnetID, privateSubnetID string) (string, error) {
	log.Infof("No route %v table found for VPCID %v, creating new route table", privacy, vpcID)
	out, err := conn.ec2.CreateRouteTable(&_ec2.CreateRouteTableInput{
		VpcId: types.StringP(vpcID),
	})
	log.Infof("Created %v route table", privacy)
	if err != nil {
		return "", err
	}

	time.Sleep(preTagDelay)

	id := *out.RouteTable.RouteTableId

	tags := map[string]string{
		"Name": fmt.Sprintf("%s-rt-%s", conn.Cluster.Name, privacy),
		"kubernetes.io/cluster/" + conn.Cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
	}

	if privacy == "public" {
		tags["sigs.k8s.io/cluster-api-provider-aws/role"] = bastion
	} else {
		tags["sigs.k8s.io/cluster-api-provider-aws/role"] = "common"
	}

	if err := conn.addTags(id, tags); err != nil {
		return "", nil
	}

	if privacy == "public" {
		_, err = conn.ec2.CreateRoute(&ec2.CreateRouteInput{
			RouteTableId:         out.RouteTable.RouteTableId,
			DestinationCidrBlock: types.StringP("0.0.0.0/0"),
			GatewayId:            types.StringP(igwID),
		})
		if err != nil {
			return "", errors.Wrapf(err, "failed to create public route")
		}

		_, err = conn.ec2.AssociateRouteTable(&ec2.AssociateRouteTableInput{
			RouteTableId: out.RouteTable.RouteTableId,
			SubnetId:     types.StringP(publicSubnetID),
		})

		if err != nil {
			return "", errors.Wrapf(err, "failed to associate route table to subnet")
		}
	} else {
		_, err = conn.ec2.CreateRoute(&ec2.CreateRouteInput{
			RouteTableId:         out.RouteTable.RouteTableId,
			DestinationCidrBlock: types.StringP("0.0.0.0/0"),
			GatewayId:            types.StringP(natID),
		})
		if err != nil {
			return "", errors.Wrapf(err, "failed to create private route")
		}

		_, err := conn.ec2.AssociateRouteTable(&ec2.AssociateRouteTableInput{
			RouteTableId: out.RouteTable.RouteTableId,
			SubnetId:     types.StringP(privateSubnetID),
		})

		if err != nil {
			return "", errors.Wrapf(err, "failed to associate route table to subnet %q", privateSubnetID)
		}
	}
	log.Infof("Route added to route table %v", id)

	return id, nil
}

func (conn *cloudConnector) setupSecurityGroups(vpcID string) error {
	var ok bool
	var err error
	if conn.Cluster.Status.Cloud.AWS.MasterSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName); !ok {
		if err != nil {
			return err
		}
		err = conn.createSecurityGroup(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName, "controlplane")
		if err != nil {
			return err
		}
		log.Infof("Master security group %v created", conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName)
	}
	if conn.Cluster.Status.Cloud.AWS.NodeSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.NodeSGName); !ok {
		if err != nil {
			return err
		}
		err = conn.createSecurityGroup(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.NodeSGName, "node")
		if err != nil {
			return err
		}
		log.Infof("Node security group %v created", conn.Cluster.Spec.Config.Cloud.AWS.NodeSGName)
	}
	if conn.Cluster.Status.Cloud.AWS.BastionSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.BastionSGName); !ok {
		if err != nil {
			return err
		}
		err = conn.createSecurityGroup(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.BastionSGName, bastion)
		if err != nil {
			return err
		}
		log.Infof("Bastion security group %v created", conn.Cluster.Spec.Config.Cloud.AWS.BastionSGName)
	}

	err = conn.detectSecurityGroups(vpcID)
	if err != nil {
		return err
	}

	roles := []string{
		bastion,
		"controlplane",
		"node",
	}

	for _, role := range roles {
		rules, err := conn.getIngressRules(role)
		if err != nil {
			return err
		}

		groupID, err := conn.getGroupID(role)
		if err != nil {
			return err
		}

		input := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: types.StringP(groupID),
		}

		input.IpPermissions = append(input.IpPermissions, rules...)

		_, err = conn.ec2.AuthorizeSecurityGroupIngress(input)
		if err != nil {
			return err
		}
	}

	return nil
}

func (conn *cloudConnector) getGroupID(role string) (string, error) {
	switch role {
	case bastion:
		return conn.Cluster.Status.Cloud.AWS.BastionSGId, nil
	case "controlplane":
		return conn.Cluster.Status.Cloud.AWS.MasterSGId, nil
	case "node":
		return conn.Cluster.Status.Cloud.AWS.NodeSGId, nil
	}
	return "", errors.New("error")
}

// ref: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/c46b0c4de65b97f0043821524f876715d9a65f0c/pkg/cloud/aws/services/ec2/securitygroups.go#L232
func (conn *cloudConnector) getIngressRules(role string) ([]*ec2.IpPermission, error) {
	switch role {
	case bastion:
		return []*ec2.IpPermission{
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(22),
				ToPort:     types.Int64P(22),
				IpRanges: []*ec2.IpRange{
					{
						Description: types.StringP("SSH"),
						CidrIp:      types.StringP("0.0.0.0/0"),
					},
				},
			},
		}, nil
	case "controlplane":
		return []*ec2.IpPermission{
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(22),
				ToPort:     types.Int64P(22),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: types.StringP("SSH"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.BastionSGId),
					},
				},
			},
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(6443),
				ToPort:     types.Int64P(6443),
				IpRanges: []*ec2.IpRange{
					{
						Description: types.StringP("Kubernetes API"),
						CidrIp:      types.StringP("0.0.0.0/0"),
					},
				},
			},
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(2379),
				ToPort:     types.Int64P(2379),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: types.StringP("etcd"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.MasterSGId),
					},
				},
			},
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(2380),
				ToPort:     types.Int64P(2380),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: types.StringP("etcd peer"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.MasterSGId),
					},
				},
			},
			{
				IpProtocol: types.StringP("tcp"), //v1alpha1.SecurityGroupProtocolTCP,
				FromPort:   types.Int64P(179),
				ToPort:     types.Int64P(179),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: types.StringP("bgp (calico)"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.MasterSGId),
					},
					{
						Description: types.StringP("bgp (calico)"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.NodeSGId),
					},
				},
			},
		}, nil

	case "node":
		return []*ec2.IpPermission{
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(22),
				ToPort:     types.Int64P(22),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: types.StringP("SSH"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.BastionSGId),
					},
				},
			},
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(30000),
				ToPort:     types.Int64P(32767),
				IpRanges: []*ec2.IpRange{
					{
						Description: types.StringP("Node Port Services"),
						CidrIp:      types.StringP("0.0.0.0/0"),
					},
				}, //[]string{anyIPv4CidrBlock},
			},
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(10250),
				ToPort:     types.Int64P(10250),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: types.StringP("Kubelet API"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.MasterSGId),
					},
				},
			},
			{
				IpProtocol: types.StringP("tcp"),
				FromPort:   types.Int64P(179),
				ToPort:     types.Int64P(179),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: types.StringP("bgp (calico)"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.MasterSGId),
					},
					{
						Description: types.StringP("bgp (calico)"),
						GroupId:     types.StringP(conn.Cluster.Status.Cloud.AWS.NodeSGId),
					},
				},
			},
		}, nil
	}

	return nil, errors.Errorf("Cannot determine ingress rules for unknown security group role %q", role)
}

func (conn *cloudConnector) getBastion() (bool, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   types.StringP("tag:Name"),
				Values: types.StringPSlice([]string{conn.namer.BastionName()}),
			},
			{
				Name:   types.StringP("tag:sigs.k8s.io/cluster-api-provider-aws/role"),
				Values: types.StringPSlice([]string{bastion}),
			},
			{
				Name:   types.StringP("instance-state-name"),
				Values: types.StringPSlice([]string{"running"}),
			},
		},
	}
	out, err := conn.ec2.DescribeInstances(input)
	if err != nil {
		return false, errors.Wrap(err, "failed to describe bastion host")
	}

	for _, res := range out.Reservations {
		for _, instance := range res.Instances {
			if aws.StringValue(instance.State.Name) != ec2.InstanceStateNameTerminated {
				//conn.Cluster.Status.Cloud.AWS.BastionID = *instance.InstanceId
				return true, nil
			}
		}
	}

	return false, errors.New("bastion host not found")
}

func (conn *cloudConnector) setupBastion(publicSubnetID string) error {
	sshKeyName := conn.Cluster.Spec.Config.Cloud.SSHKeyName
	name := fmt.Sprintf("%s-bastion", conn.Cluster.Name)

	userData, err := clusterapi_aws.NewBastion(&clusterapi_aws.BastionInput{})
	if err != nil {
		return err
	}

	input := &ec2.RunInstancesInput{
		InstanceType: types.StringP("t2.micro"),
		SubnetId:     types.StringP(publicSubnetID),
		ImageId:      types.StringP(getBastionAMI(conn.Cluster.Spec.Config.Cloud.Region)),
		KeyName:      &sshKeyName,
		MaxCount:     types.Int64P(1),
		MinCount:     types.Int64P(1),
		UserData:     types.StringP(base64.StdEncoding.EncodeToString([]byte(userData))),
		SecurityGroupIds: []*string{
			types.StringP(conn.Cluster.Status.Cloud.AWS.BastionSGId),
		},
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: types.StringP("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   types.StringP("Name"),
						Value: types.StringP(name),
					},
					{
						Key:   types.StringP("sigs.k8s.io/cluster-api-provider-aws/role"),
						Value: types.StringP(bastion),
					},
					{
						Key:   types.StringP("kubernetes.io/cluster/" + conn.Cluster.Name),
						Value: types.StringP("owned"),
					},
					{
						Key:   types.StringP("sigs.k8s.io/cluster-api-provider-aws/managed"),
						Value: types.StringP("true"),
					},
				},
			},
		},
	}

	out, err := conn.ec2.RunInstances(input)
	if err != nil {
		return errors.Wrapf(err, "failed to run instance")
	}

	if len(out.Instances) == 0 {
		return errors.Errorf("no instance returned for reservation %v", out.GoString())
	}

	return nil
}

// ref: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/27e2e7fec56549f0dd0ff41b12d113ed94b2bad1/pkg/cloud/aws/services/ec2/ami.go#L128
func getBastionAMI(region string) string {
	switch region {
	case "ap-northeast-1":
		return "ami-d39a02b5"
	case "ap-northeast-2":
		return "ami-67973709"
	case "ap-south-1":
		return "ami-5d055232"
	case "ap-southeast-1":
		return "ami-325d2e4e"
	case "ap-southeast-2":
		return "ami-37df2255"
	case "ca-central-1":
		return "ami-f0870294"
	case "eu-central-1":
		return "ami-af79ebc0"
	case "eu-west-1":
		return "ami-4d46d534"
	case "eu-west-2":
		return "ami-d7aab2b3"
	case "eu-west-3":
		return "ami-5e0eb923"
	case "sa-east-1":
		return "ami-1157157d"
	case "us-east-1":
		return "ami-41e0b93b"
	case "us-east-2":
		return "ami-2581aa40"
	case "us-west-1":
		return "ami-79aeae19"
	case "us-west-2":
		return "ami-1ee65166"
	default:
		return "unknown region"
	}
}

func (conn *cloudConnector) getSecurityGroupID(vpcID, groupName string) (string, bool, error) {
	log.Infof("Checking security group %v", groupName)
	r1, err := conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(vpcID),
				},
			},
			{
				Name: types.StringP("group-name"),
				Values: []*string{
					types.StringP(groupName),
				},
			},
		},
	})
	log.Debug("Retrieved security group", r1, err)
	if err != nil {
		return "", false, err
	}
	if len(r1.SecurityGroups) == 0 {
		log.Infof("No security group %v found", groupName)
		return "", false, nil
	}
	log.Infof("Security group %v found", groupName)
	return *r1.SecurityGroups[0].GroupId, true, nil
}

func (conn *cloudConnector) createSecurityGroup(vpcID, groupName string, instanceType string) error {
	log.Infof("Creating security group %v", groupName)
	r2, err := conn.ec2.CreateSecurityGroup(&_ec2.CreateSecurityGroupInput{
		GroupName:   types.StringP(groupName),
		Description: types.StringP("kubernetes security group for " + instanceType),
		VpcId:       types.StringP(vpcID),
	})
	log.Debug("Created security group", r2, err)
	if err != nil {
		return err
	}

	time.Sleep(preTagDelay)

	tags := map[string]string{
		"Name": conn.Cluster.Name + "-" + instanceType,
		"kubernetes.io/cluster/" + conn.Cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    instanceType,
	}

	return conn.addTags(*r2.GroupId, tags)
}

func (conn *cloudConnector) detectSecurityGroups(vpcID string) error {
	var ok bool
	var err error
	if conn.Cluster.Status.Cloud.AWS.MasterSGId == "" {
		if conn.Cluster.Status.Cloud.AWS.MasterSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName); !ok {
			return errors.Errorf("[%s] could not detect Kubernetes master security group.  Make sure you've launched a cluster with appctl", "")
		}
		log.Infof("Master security group %v with id %v detected", conn.Cluster.Spec.Config.Cloud.AWS.MasterSGName, conn.Cluster.Status.Cloud.AWS.MasterSGId)

	}
	if conn.Cluster.Status.Cloud.AWS.NodeSGId == "" {
		if conn.Cluster.Status.Cloud.AWS.NodeSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.NodeSGName); !ok {
			return errors.Errorf("[%s] could not detect Kubernetes node security group.  Make sure you've launched a cluster with appctl", "")
		}
		log.Infof("Node security group %v with id %v detected", conn.Cluster.Spec.Config.Cloud.AWS.NodeSGName, conn.Cluster.Status.Cloud.AWS.NodeSGId)
	}
	if conn.Cluster.Status.Cloud.AWS.BastionSGId == "" {
		if conn.Cluster.Status.Cloud.AWS.BastionSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.Cluster.Spec.Config.Cloud.AWS.BastionSGName); !ok {
			return errors.Errorf("[%s] could not detect Kubernetes bastion security group.  Make sure you've launched a cluster with appctl", "")
		}
		log.Infof("Bastion security group %v with id %v detected", conn.Cluster.Spec.Config.Cloud.AWS.BastionSGName, conn.Cluster.Status.Cloud.AWS.BastionSGId)
	}
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) getMaster(name string) (bool, error) {
	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name:   types.StringP("tag:Name"),
				Values: types.StringPSlice([]string{name}),
			},
			{
				Name:   types.StringP("tag:sigs.k8s.io/cluster-api-provider-aws/role"),
				Values: types.StringPSlice([]string{"controlplane"}),
			},
			{
				Name:   types.StringP("instance-state-name"),
				Values: types.StringPSlice([]string{"running"}),
			},
		},
	})
	if err != nil {
		return false, err
	}
	if len(r1.Reservations) == 0 {
		return false, nil
	}
	return true, err
}

func (conn *cloudConnector) startMaster(machine *clusterapi.Machine, privateSubnetID, script string) (*ec2.Instance, error) {
	sshKeyName := conn.Cluster.Spec.Config.Cloud.SSHKeyName

	if err := conn.detectUbuntuImage(); err != nil {
		return nil, err
	}

	providerSpec, err := clusterapi_aws.MachineConfigFromProviderSpec(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}

	input := &ec2.RunInstancesInput{
		InstanceType: types.StringP(providerSpec.InstanceType),
		SubnetId:     types.StringP(privateSubnetID),
		ImageId:      types.StringP(conn.Cluster.Spec.Config.Cloud.InstanceImage),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: types.StringP(conn.Cluster.Spec.Config.Cloud.AWS.IAMProfileMaster),
		},
		KeyName:  &sshKeyName,
		MaxCount: types.Int64P(1),
		MinCount: types.Int64P(1),
		UserData: types.StringP(base64.StdEncoding.EncodeToString([]byte(script))),
		SecurityGroupIds: []*string{
			types.StringP(conn.Cluster.Status.Cloud.AWS.MasterSGId),
		},

		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: types.StringP("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   types.StringP("Name"),
						Value: types.StringP(machine.Name),
					},
					{
						Key:   types.StringP("sigs.k8s.io/cluster-api-provider-aws/role"),
						Value: types.StringP(machine.Labels["set"]),
					},
					{
						Key:   types.StringP("kubernetes.io/cluster/" + conn.Cluster.Name),
						Value: types.StringP("owned"),
					},
					{
						Key:   types.StringP("sigs.k8s.io/cluster-api-provider-aws/managed"),
						Value: types.StringP("true"),
					},
				},
			},
		},
	}

	out, err := conn.ec2.RunInstances(input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run instance")
	}

	if len(out.Instances) == 0 {
		return nil, errors.Errorf("no instance returned for reservation %v", out.GoString())
	}

	masterInstance := out.Instances[0]

	log.Info("Waiting for master instance to be ready")
	// We are not able to add an elastic ip, a route or volume to the instance until that instance is in "running" state.
	err = conn.waitForInstanceState(*masterInstance.InstanceId, "running")
	if err != nil {
		return nil, err
	}
	log.Info("Master instance is ready")

	lbinput := &elb.RegisterInstancesWithLoadBalancerInput{
		Instances:        []*elb.Instance{{InstanceId: masterInstance.InstanceId}},
		LoadBalancerName: types.StringP(fmt.Sprintf("%s-apiserver", conn.Cluster.Name)),
	}

	if _, err := conn.elb.RegisterInstancesWithLoadBalancer(lbinput); err != nil {
		return nil, err
	}

	r, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		InstanceIds: []*string{masterInstance.InstanceId},
	})
	if err != nil {
		return nil, err
	}

	return r.Reservations[0].Instances[0], nil
}

func (conn *cloudConnector) waitForInstanceState(instanceID, state string) error {
	for {
		r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
			InstanceIds: []*string{types.StringP(instanceID)},
		})
		if err != nil {
			return err
		}
		curState := *r1.Reservations[0].Instances[0].State.Name
		if curState == state {
			break
		}
		log.Infof("Waiting for instance %v to be %v (currently %v)", instanceID, state, curState)
		log.Infof("Sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (conn *cloudConnector) allocateElasticIP() (string, string, error) {
	r1, err := conn.ec2.AllocateAddress(&_ec2.AllocateAddressInput{
		Domain: types.StringP("vpc"),
	})
	log.Debug("Allocated elastic IP", r1, err)
	if err != nil {
		return "", "", err
	}
	log.Infof("Elastic IP %v allocated", *r1.PublicIp)
	time.Sleep(5 * time.Second)

	tags := map[string]string{
		"Name": conn.Cluster.Name + "-eip-apiserver",
		"kubernetes.io/cluster/" + conn.Cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    "apiserver",
	}

	if err := conn.addTags(*r1.AllocationId, tags); err != nil {
		return "", "", err
	}

	return *r1.AllocationId, *r1.PublicIp, nil
}

func (conn *cloudConnector) findElasticIP() ([]*ec2.Address, error) {
	out, err := conn.ec2.DescribeAddresses(&ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{
				Name: types.StringP("tag:Name"),
				Values: []*string{
					types.StringP(conn.Cluster.Name + "-eip-apiserver"),
				},
			},
			{
				Name: types.StringP("tag-key"),
				Values: []*string{
					types.StringP("kubernetes.io/cluster/" + conn.Cluster.Name),
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return out.Addresses, nil
}

func (conn *cloudConnector) releaseReservedIP() error {
	ips, err := conn.findElasticIP()
	if err != nil {
		return err
	}

	if len(ips) == 0 {
		return nil
	}

	if _, err := conn.ec2.ReleaseAddress(&ec2.ReleaseAddressInput{
		AllocationId: ips[0].AllocationId,
	}); err != nil {
		return err
	}

	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (done bool, err error) {
		log.Infoln("waiting for elastic ip to be released")
		ips, err := conn.findElasticIP()
		if err != nil {
			return false, nil
		}
		if len(ips) == 0 {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) deleteSecurityGroup(vpcID string) error {
	log.Infof("deleting security group")

	if vpcID == "" {
		log.Infof("vpc-id is empty, vpc already deleted")
		return nil
	}

	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (done bool, err error) {
		r, err := conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
			Filters: []*_ec2.Filter{
				{
					Name: types.StringP("vpc-id"),
					Values: []*string{
						types.StringP(vpcID),
					},
				},
				{
					Name: types.StringP("tag-key"),
					Values: []*string{
						types.StringP("kubernetes.io/cluster/" + conn.Cluster.Name),
					},
				},
			},
		})
		if err != nil {
			log.Infof("failed to list securit groups: %v", err)
			return false, nil
		}

		for _, sg := range r.SecurityGroups {
			if len(sg.IpPermissions) > 0 {
				_, err := conn.ec2.RevokeSecurityGroupIngress(&_ec2.RevokeSecurityGroupIngressInput{
					GroupId:       sg.GroupId,
					IpPermissions: sg.IpPermissions,
				})
				if err != nil {
					log.Infof("failed to delete security group IpPermissions: %v", err)
					return false, nil
				}
			}

			if len(sg.IpPermissionsEgress) > 0 {
				_, err := conn.ec2.RevokeSecurityGroupEgress(&_ec2.RevokeSecurityGroupEgressInput{
					GroupId:       sg.GroupId,
					IpPermissions: sg.IpPermissionsEgress,
				})
				if err != nil {
					log.Infof("failed to delete security group IpPermissionsEgress: %v", err)
					return false, nil
				}
			}
		}

		for _, sg := range r.SecurityGroups {
			_, err := conn.ec2.DeleteSecurityGroup(&_ec2.DeleteSecurityGroupInput{
				GroupId: sg.GroupId,
			})
			if err != nil {
				log.Infof("failed to delete security group %v: %v", sg.GroupName, err)
				return false, nil
			}
		}

		log.Infof("successfully deleted security groups")
		return true, nil
	})
}

func (conn *cloudConnector) deleteSubnetID(vpcID string) error {
	r, err := conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(vpcID),
				},
			},
			{
				Name: types.StringP("tag-key"),
				Values: []*string{
					types.StringP("kubernetes.io/cluster/" + conn.Cluster.Name),
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
		log.Infof("Subnet ID in VPC %v is deleted", *subnet.SubnetId)
	}
	return nil
}

func (conn *cloudConnector) deleteInternetGateway(vpcID string) error {
	r1, err := conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("attachment.vpc-id"),
				Values: []*string{
					types.StringP(vpcID),
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
			VpcId:             types.StringP(vpcID),
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
	log.Infof("Internet gateway for cluster %v are deleted", conn.Cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteRouteTable(vpcID string) error {
	r1, err := conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("vpc-id"),
				Values: []*string{
					types.StringP(vpcID),
				},
			},
			{
				Name: types.StringP("tag-key"),
				Values: []*string{
					types.StringP("kubernetes.io/cluster/" + conn.Cluster.Name),
				},
			},
		},
	})

	if err != nil {
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
					return err
				}
			}
		}
		if !mainTable {
			_, err := conn.ec2.DeleteRouteTable(&_ec2.DeleteRouteTableInput{
				RouteTableId: rt.RouteTableId,
			})
			if err != nil {
				return err
			}
		}
	}
	log.Infof("Route tables for cluster %v are deleted", conn.Cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteNatGateway(natID string) error {
	if natID == "" {
		log.Infof("NAT already deleted, exiting")
		return nil
	}

	if _, err := conn.ec2.DeleteNatGateway(&ec2.DeleteNatGatewayInput{
		NatGatewayId: types.StringP(natID),
	}); err != nil {
		return err
	}

	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		log.Infoln("waiting for nat to be deleted")
		out, err := conn.ec2.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{
				types.StringP(natID),
			},
		})
		if err != nil {
			return false, nil
		}
		if len(out.NatGateways) == 0 {
			return true, nil
		}

		if *out.NatGateways[0].State == "deleted" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) deleteVpc(vpcID string) error {
	_, err := conn.ec2.DeleteVpc(&_ec2.DeleteVpcInput{
		VpcId: types.StringP(vpcID),
	})

	if err != nil {
		return err
	}
	log.Infof("VPC for cluster %v is deleted", conn.Cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteSSHKey() error {
	var err error
	_, err = conn.ec2.DeleteKeyPair(&_ec2.DeleteKeyPairInput{
		KeyName: types.StringP(conn.Cluster.Spec.Config.Cloud.SSHKeyName),
	})
	if err != nil {
		return err
	}
	log.Infof("SSH key for cluster %v is deleted", conn.Cluster.Name)

	return err
}

func (conn *cloudConnector) deleteInstance(role string) error {
	log.Infof("deleting instance %s", role)
	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("tag:sigs.k8s.io/cluster-api-provider-aws/role"),
				Values: []*string{
					types.StringP(role),
				},
			},
			{
				Name: types.StringP("tag-key"),
				Values: []*string{
					types.StringP(fmt.Sprintf("kubernetes.io/cluster/%s", conn.Cluster.Name)),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	instances := make([]*string, 0)
	for _, reservation := range r1.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, instance.InstanceId)
		}
	}

	if len(instances) == 0 {
		return nil
	}

	fmt.Printf("TerminateInstances %v", instances)
	log.Infof("Terminating %v instance for cluster %v", role, conn.Cluster.Name)
	_, err = conn.ec2.TerminateInstances(&_ec2.TerminateInstancesInput{
		InstanceIds: instances,
	})
	if err != nil {
		return err
	}
	instanceInput := &_ec2.DescribeInstancesInput{
		InstanceIds: instances,
	}
	err = conn.ec2.WaitUntilInstanceTerminated(instanceInput)
	log.Infof("%v instance for cluster %v is terminated", role, conn.Cluster.Name)
	return err
}
