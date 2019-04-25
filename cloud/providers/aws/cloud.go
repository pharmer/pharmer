package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"
	"time"

	. "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	. "github.com/appscode/go/types"
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
	"github.com/aws/aws-sdk-go/service/sts"
	awssts "github.com/aws/aws-sdk-go/service/sts"
	clusterapi_aws "github.com/pharmer/pharmer/apis/v1beta1/aws"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	//_ "github.com/aws/aws-sdk-go/service/lightsail"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
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

var _ ClusterApiProviderComponent = &cloudConnector{}

func NewConnector(cm *ClusterManager) (*cloudConnector, error) {
	cred, err := Store(cm.ctx).Owner(cm.owner).Credentials().Get(cm.cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.AWS{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cm.cluster.Spec.Config.CredentialName, err)
	}

	config := &_aws.Config{
		Region:      &cm.cluster.Spec.Config.Cloud.Region,
		Credentials: credentials.NewStaticCredentials(typed.AccessKeyID(), typed.SecretAccessKey(), ""),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}
	conn := cloudConnector{
		ctx:       cm.ctx,
		cluster:   cm.cluster,
		ec2:       _ec2.New(sess),
		elb:       _elb.New(sess),
		iam:       _iam.New(sess),
		autoscale: autoscaling.New(sess),
		s3:        _s3.New(sess),
		namer:     cm.namer,
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		return nil, errors.Errorf("credential %s does not have necessary authorization. Reason: %s", cm.cluster.Spec.Config.CredentialName, msg)
	}
	return &conn, nil
}

func PrepareCloud(cm *ClusterManager) error {
	var err error

	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
		return err
	}
	if cm.ctx, err = LoadEtcdCertificate(cm.ctx, cm.cluster, cm.owner); err != nil {
		return err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return err
	}
	if cm.ctx, err = LoadSaKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return err
	}

	if cm.conn, err = NewConnector(cm); err != nil {
		return err
	}

	return nil
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
	conn.cluster.Spec.Config.Cloud.OS = "ubuntu"
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
	conn.cluster.Spec.Config.Cloud.InstanceImage = *r1.Images[0].ImageId
	Logger(conn.ctx).Infof("Ubuntu image with %v detected", conn.cluster.Spec.Config.Cloud.InstanceImage)
	return nil
}

func (conn *cloudConnector) CreateCredentialSecret(kc kubernetes.Interface, data map[string]string) error {
	err := CreateNamespace(kc, "aws-provider-system")
	if err != nil {
		return err
	}

	credTemplate := template.Must(template.New("aws-cred").Parse(
		`[default]
aws_access_key_id = {{ .AccessKeyID }}
aws_secret_access_key = {{ .SecretAccessKey }}
region = {{ .Region }}
`))

	var buf bytes.Buffer
	err = credTemplate.Execute(&buf, struct {
		AccessKeyID     string
		SecretAccessKey string
		Region          string
	}{
		AccessKeyID:     data["accessKeyID"],
		SecretAccessKey: data["secretAccessKey"],
		Region:          conn.cluster.Spec.Config.Cloud.Region,
	})
	if err != nil {
		return err
	}

	credData := buf.Bytes()

	if err = CreateSecret(kc, "aws-provider-manager-bootstrap-credentials-kt5bhb6h9c", "aws-provider-system", map[string][]byte{
		"credentials": credData,
	}); err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) getIAMProfile() (bool, error) {
	r1, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.cluster.Spec.Config.Cloud.AWS.IAMProfileMaster})
	if r1.InstanceProfile == nil {
		return false, err
	}
	r2, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.cluster.Spec.Config.Cloud.AWS.IAMProfileNode})
	if r2.InstanceProfile == nil {
		return false, err
	}
	return true, nil
}

func (conn *cloudConnector) ensureIAMProfile() error {
	r1, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.cluster.Spec.Config.Cloud.AWS.IAMProfileMaster})
	if err != nil && !strings.Contains(err.Error(), "NoSuchEntity") {
		return err
	}
	if r1.InstanceProfile == nil {
		err := conn.createIAMProfile(api.RoleMaster, conn.cluster.Spec.Config.Cloud.AWS.IAMProfileMaster)
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Master instance profile %v created", conn.cluster.Spec.Config.Cloud.AWS.IAMProfileMaster)
	}
	r2, err := conn.iam.GetInstanceProfile(&_iam.GetInstanceProfileInput{InstanceProfileName: &conn.cluster.Spec.Config.Cloud.AWS.IAMProfileNode})
	if err != nil && !strings.Contains(err.Error(), "NoSuchEntity") {
		return err
	}
	if r2.InstanceProfile == nil {
		err := conn.createIAMProfile(api.RoleNode, conn.cluster.Spec.Config.Cloud.AWS.IAMProfileNode)
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Node instance profile %v created", conn.cluster.Spec.Config.Cloud.AWS.IAMProfileNode)
	}
	return nil
}

func (conn *cloudConnector) deleteIAMProfile() error {
	if err := conn.deleteRolePolicy(conn.cluster.Spec.Config.Cloud.AWS.IAMProfileMaster); err != nil {
		Logger(conn.ctx).Infoln("Failed to delete IAM instance-policy ", conn.cluster.Spec.Config.Cloud.AWS.IAMProfileMaster, err)
	}
	if err := conn.deleteRolePolicy(conn.cluster.Spec.Config.Cloud.AWS.IAMProfileNode); err != nil {
		Logger(conn.ctx).Infoln("Failed to delete IAM instance-policy ", conn.cluster.Spec.Config.Cloud.AWS.IAMProfileNode, err)
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

	if role == conn.cluster.Spec.Config.Cloud.AWS.IAMProfileMaster {
		if _, err := conn.iam.DeleteRolePolicy(&_iam.DeleteRolePolicyInput{
			PolicyName: &conn.cluster.Spec.Config.Cloud.AWS.IAMProfileNode,
			RoleName:   &role,
		}); err != nil {
			Logger(conn.ctx).Infoln("Failed to delete role policy", role, err)
		}
		if _, err := conn.iam.DeleteRolePolicy(&_iam.DeleteRolePolicyInput{
			PolicyName: StringP(conn.namer.ControlPlanePolicyName()),
			RoleName:   &role,
		}); err != nil {
			Logger(conn.ctx).Infoln("Failed to delete role policy", role, err)
		}
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

	if role == api.RoleMaster {
		controlPlanePolicy := &_iam.PutRolePolicyInput{
			RoleName:       &key,
			PolicyName:     &key,
			PolicyDocument: StringP(strings.TrimSpace(IAMMasterPolicy)),
		}
		if _, err := conn.iam.PutRolePolicy(controlPlanePolicy); err != nil {
			log.Debugf("Error attaching control-plane policy to control-plane instance profile")
			return err
		}

		controllerPolicy := &_iam.PutRolePolicyInput{
			RoleName:       &key,
			PolicyName:     StringP(strings.Replace(key, "master", "controller", 1)),
			PolicyDocument: StringP(strings.TrimSpace(strings.Replace(IAMControllerPolicy, "ACCOUNT_ID", *accountID.Account, -1))),
		}
		if _, err := conn.iam.PutRolePolicy(controllerPolicy); err != nil {
			log.Debugf("Error attaching controller policy to control-plane instance profile")
			return err
		}

		nodePolicy := &_iam.PutRolePolicyInput{
			RoleName:       &key,
			PolicyName:     StringP(strings.Replace(key, "master", "node", 1)),
			PolicyDocument: StringP(strings.TrimSpace(IAMNodePolicy)),
		}
		if _, err := conn.iam.PutRolePolicy(nodePolicy); err != nil {
			log.Debugf("Error attaching node policy to control-plane instance profile")
			return err
		}
	} else {
		nodePolicy := &_iam.PutRolePolicyInput{
			RoleName:       &key,
			PolicyName:     &key,
			PolicyDocument: StringP(strings.TrimSpace(IAMNodePolicy)),
		}
		if _, err := conn.iam.PutRolePolicy(nodePolicy); err != nil {
			log.Debugf("Error attaching node policy to node instance profile")
			return err
		}
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
		KeyNames: StringPSlice([]string{conn.cluster.Spec.Config.Cloud.SSHKeyName}),
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
		KeyName:           StringP(conn.cluster.Spec.Config.Cloud.SSHKeyName),
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

func (conn *cloudConnector) getVpc() (string, bool, error) {
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
				Name: StringP("tag-key"),
				Values: []*string{
					StringP("kubernetes.io/cluster/" + conn.cluster.Name),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("VPC described", r1, err)
	if len(r1.Vpcs) > 0 {
		Logger(conn.ctx).Infof("VPC %v found", *r1.Vpcs[0].VpcId)
		return *r1.Vpcs[0].VpcId, true, nil
	}

	return "", false, errors.New("VPC not found")
}

func (conn *cloudConnector) setupVpc() (string, error) {
	Logger(conn.ctx).Info("No VPC found, creating new VPC")
	r2, err := conn.ec2.CreateVpc(&_ec2.CreateVpcInput{
		CidrBlock: StringP(conn.cluster.Spec.Config.Cloud.AWS.VpcCIDR),
	})
	Logger(conn.ctx).Debug("VPC created", r2, err)

	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("VPC %v created", *r2.Vpc.VpcId)

	wReq := &ec2.DescribeVpcsInput{VpcIds: []*string{r2.Vpc.VpcId}}
	if err := conn.ec2.WaitUntilVpcAvailable(wReq); err != nil {
		return "", errors.Wrapf(err, "failed to wait for vpc %q", *r2.Vpc.VpcId)
	}

	tags := map[string]string{
		"Name": conn.cluster.Name + "-vpc",
		"kubernetes.io/cluster/" + conn.cluster.Name:   "owned",
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

func (conn *cloudConnector) getLoadBalancer() (string, error) {
	input := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: aws.StringSlice([]string{fmt.Sprintf("%s-apiserver", conn.cluster.Name)}),
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
		LoadBalancerName: StringP(fmt.Sprintf("%s-apiserver", conn.cluster.Name)),
	}
	if _, err := conn.elb.DeleteLoadBalancer(input); err != nil {
		return false, err
	}

	if err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
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
		LoadBalancerName: StringP(fmt.Sprintf("%s-apiserver", conn.cluster.Name)),
		Subnets:          []*string{StringP(publicSubnetID)},
		SecurityGroups:   []*string{StringP(conn.cluster.Status.Cloud.AWS.MasterSGId)},
		Scheme:           StringP("Internet-facing"),
		Listeners: []*elb.Listener{
			{
				Protocol:         StringP("TCP"),
				LoadBalancerPort: Int64P(6443),
				InstanceProtocol: StringP("TCP"),
				InstancePort:     Int64P(6443),
			},
		},
		Tags: []*elb.Tag{
			{
				Key:   StringP("sigs.k8s.io/cluster-api-provider-aws/role"),
				Value: StringP("apiserver"),
			},
			{
				Key:   StringP(fmt.Sprintf("kubernetes.io/cluster/%s", conn.cluster.Name)),
				Value: StringP("owned"),
			},
			{
				Key:   StringP("sigs.k8s.io/cluster-api-provider-aws/managed"),
				Value: StringP("true"),
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
			Target:             StringP("TCP:6443"),
			Interval:           Int64P(int64((10 * time.Second).Seconds())),
			Timeout:            Int64P(int64((5 * time.Second).Seconds())),
			HealthyThreshold:   Int64P(5),
			UnhealthyThreshold: Int64P(3),
		},
	}

	if _, err := conn.elb.ConfigureHealthCheck(hc); err != nil {
		return "", err
	}

	return *output.DNSName, nil
}

func (conn *cloudConnector) getSubnet(vpcID, privacy string) (string, bool, error) {
	Logger(conn.ctx).Infof("Checking for existing %s subnet", privacy)
	r1, err := conn.ec2.DescribeSubnets(&_ec2.DescribeSubnetsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("availabilityZone"),
				Values: []*string{
					StringP(conn.cluster.Spec.Config.Cloud.Zone),
				},
			},
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(vpcID),
				},
			},
			{
				Name: StringP("tag:Name"),
				Values: []*string{
					StringP(fmt.Sprintf("%s-subnet-%s", conn.cluster.Name, privacy)),
				},
			},
		},
	})
	Logger(conn.ctx).Debug(fmt.Sprintf("Retrieved %s subnet", privacy), r1, err)
	if err != nil {
		return "", false, err
	}

	if len(r1.Subnets) == 0 {
		return "", false, errors.Errorf("No %s subnet found", privacy)
	}
	Logger(conn.ctx).Infof("%s Subnet %v found with CIDR %v", privacy, r1.Subnets[0].SubnetId, r1.Subnets[0].CidrBlock)

	return *r1.Subnets[0].SubnetId, true, nil

}

func (conn *cloudConnector) setupPrivateSubnet(vpcID string) (string, error) {
	Logger(conn.ctx).Info("No subnet found, creating new subnet")
	r2, err := conn.ec2.CreateSubnet(&_ec2.CreateSubnetInput{
		CidrBlock:        StringP(conn.cluster.Spec.Config.Cloud.AWS.PrivateSubnetCIDR),
		VpcId:            StringP(vpcID),
		AvailabilityZone: StringP(conn.cluster.Spec.Config.Cloud.Zone),
	})
	Logger(conn.ctx).Debug("Created subnet", r2, err)
	if err != nil {
		return "", err
	}

	id := *r2.Subnet.SubnetId
	Logger(conn.ctx).Infof("Subnet %v created", id)

	time.Sleep(preTagDelay)

	tags := map[string]string{
		"Name": conn.cluster.Name + "-subnet-private",
		"kubernetes.io/cluster/" + conn.cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    "common",
	}

	if err := conn.addTags(id, tags); err != nil {
		return "", err
	}

	return id, nil
}

func (conn *cloudConnector) setupPublicSubnet(vpcID string) (string, error) {
	Logger(conn.ctx).Info("No subnet found, creating new subnet")
	r2, err := conn.ec2.CreateSubnet(&_ec2.CreateSubnetInput{
		CidrBlock:        StringP(conn.cluster.Spec.Config.Cloud.AWS.PublicSubnetCIDR),
		VpcId:            StringP(vpcID),
		AvailabilityZone: StringP(conn.cluster.Spec.Config.Cloud.Zone),
	})
	Logger(conn.ctx).Debug("Created subnet", r2, err)
	if err != nil {
		return "", err
	}
	Logger(conn.ctx).Infof("Subnet %v created", *r2.Subnet.SubnetId)

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
		"Name": conn.cluster.Name + "-subnet-public",
		"kubernetes.io/cluster/" + conn.cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    "bastion",
	}

	if err := conn.addTags(id, tags); err != nil {
		return "", err
	}

	return id, nil
}

func (conn *cloudConnector) getInternetGateway(vpcID string) (string, bool, error) {
	Logger(conn.ctx).Infof("Checking IGW with attached VPCID %v", vpcID)
	r1, err := conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("attachment.vpc-id"),
				Values: []*string{
					StringP(vpcID),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Retrieved IGW", r1, err)
	if err != nil {
		return "", false, err
	}
	if len(r1.InternetGateways) == 0 {
		return "", false, errors.Errorf("IGW not found")
	}
	Logger(conn.ctx).Infof("IGW %v found", *r1.InternetGateways[0].InternetGatewayId)
	return *r1.InternetGateways[0].InternetGatewayId, true, nil
}

func (conn *cloudConnector) setupInternetGateway(vpcID string) (string, error) {
	Logger(conn.ctx).Info("No IGW found, creating new IGW")
	r2, err := conn.ec2.CreateInternetGateway(&_ec2.CreateInternetGatewayInput{})
	Logger(conn.ctx).Debug("Created IGW", r2, err)
	if err != nil {
		return "", err
	}
	time.Sleep(preTagDelay)
	Logger(conn.ctx).Infof("IGW %v created", *r2.InternetGateway.InternetGatewayId)

	r3, err := conn.ec2.AttachInternetGateway(&_ec2.AttachInternetGatewayInput{
		InternetGatewayId: StringP(*r2.InternetGateway.InternetGatewayId),
		VpcId:             StringP(vpcID),
	})
	Logger(conn.ctx).Debug("Attached IGW to VPC", r3, err)
	if err != nil {
		return "", err
	}

	id := *r2.InternetGateway.InternetGatewayId
	Logger(conn.ctx).Infof("Attached IGW %v to VPCID %v", id, vpcID)

	tags := map[string]string{
		"Name": conn.cluster.Name + "-igw",
		"kubernetes.io/cluster/" + conn.cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    "common",
	}

	if err := conn.addTags(id, tags); err != nil {
		return "", err
	}

	return id, nil
}

func (conn *cloudConnector) getNatGateway(vpcID string) (string, bool, error) {
	Logger(conn.ctx).Infof("Checking NAT with attached VPCID %v", vpcID)
	r1, err := conn.ec2.DescribeNatGateways(&_ec2.DescribeNatGatewaysInput{
		Filter: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(vpcID),
				},
			},
			{
				Name: StringP("state"),
				Values: []*string{
					StringP("available"),
					StringP("pending"),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("Retrieved NAT", r1, err)
	if err != nil {
		return "", false, err
	}
	if len(r1.NatGateways) == 0 {
		return "", false, errors.Errorf("NAT not found")
	}

	Logger(conn.ctx).Infof("NAT %v found", *r1.NatGateways[0].NatGatewayId)
	return *r1.NatGateways[0].NatGatewayId, true, nil
}

func (conn *cloudConnector) setupNatGateway(publicSubnetID string) (string, error) {
	Logger(conn.ctx).Info("No NAT Gateway found, creating new")

	id, _, err := conn.allocateElasticIP()
	if err != nil {
		return "", err
	}

	out, err := conn.ec2.CreateNatGateway(&_ec2.CreateNatGatewayInput{
		AllocationId: StringP(id),
		SubnetId:     StringP(publicSubnetID),
	})

	wReq := &ec2.DescribeNatGatewaysInput{NatGatewayIds: []*string{out.NatGateway.NatGatewayId}}
	if err := conn.ec2.WaitUntilNatGatewayAvailable(wReq); err != nil {
		return "", errors.Wrapf(err, "failed to wait for nat gateway %q in subnet %q", *out.NatGateway.NatGatewayId, publicSubnetID)
	}

	Logger(conn.ctx).Debug("Created NAT", out, err)
	if err != nil {
		return "", err
	}

	Logger(conn.ctx).Infof("Nat Gateway %v created", *out.NatGateway.NatGatewayId)

	tags := map[string]string{
		"Name": conn.cluster.Name + "-nat",
		"kubernetes.io/cluster/" + conn.cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    "common",
	}

	if err := conn.addTags(*out.NatGateway.NatGatewayId, tags); err != nil {
		return "", err
	}

	return *out.NatGateway.NatGatewayId, nil
}

func (conn *cloudConnector) getRouteTable(privacy, vpcID string) (string, bool, error) {
	Logger(conn.ctx).Infof("Checking %v route table for VPCID %v", privacy, vpcID)
	r1, err := conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(vpcID),
				},
			},
			{
				Name: StringP("tag-key"),
				Values: []*string{
					StringP(fmt.Sprintf("kubernetes.io/cluster/%s", conn.cluster.Name)),
				},
			},
			{
				Name: StringP("tag:Name"),
				Values: []*string{
					StringP(fmt.Sprintf("%s-rt-%s", conn.cluster.Name, privacy)),
				},
			},
		},
	})
	Logger(conn.ctx).Debug("found %v route table", privacy, err)
	if err != nil {
		return "", false, err
	}
	if len(r1.RouteTables) == 0 {
		return "", false, errors.Errorf("Route table not found")
	}
	Logger(conn.ctx).Infof("%v Route table %v found", privacy, *r1.RouteTables[0].RouteTableId)

	return *r1.RouteTables[0].RouteTableId, true, nil
}

func (conn *cloudConnector) setupRouteTable(privacy, vpcID, igwID, natID, publicSubnetID, privateSubnetID string) (string, error) {
	Logger(conn.ctx).Infof("No route %v table found for VPCID %v, creating new route table", privacy, vpcID)
	out, err := conn.ec2.CreateRouteTable(&_ec2.CreateRouteTableInput{
		VpcId: StringP(vpcID),
	})
	Logger(conn.ctx).Debug("Created %v route table", privacy, out, err)
	if err != nil {
		return "", err
	}

	time.Sleep(preTagDelay)

	id := *out.RouteTable.RouteTableId

	tags := map[string]string{
		"Name": fmt.Sprintf("%s-rt-%s", conn.cluster.Name, privacy),
		"kubernetes.io/cluster/" + conn.cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
	}

	if privacy == "public" {
		tags["sigs.k8s.io/cluster-api-provider-aws/role"] = "bastion"
	} else {
		tags["sigs.k8s.io/cluster-api-provider-aws/role"] = "common"
	}

	if err := conn.addTags(id, tags); err != nil {
		return "", nil
	}

	if privacy == "public" {
		_, err = conn.ec2.CreateRoute(&ec2.CreateRouteInput{
			RouteTableId:         out.RouteTable.RouteTableId,
			DestinationCidrBlock: StringP("0.0.0.0/0"),
			GatewayId:            StringP(igwID),
		})
		if err != nil {
			return "", err
		}

		_, err = conn.ec2.AssociateRouteTable(&ec2.AssociateRouteTableInput{
			RouteTableId: out.RouteTable.RouteTableId,
			SubnetId:     StringP(publicSubnetID),
		})

		if err != nil {
			return "", errors.Wrapf(err, "failed to associate route table to subnet")
		}
	} else {
		_, err = conn.ec2.CreateRoute(&ec2.CreateRouteInput{
			RouteTableId:         out.RouteTable.RouteTableId,
			DestinationCidrBlock: StringP("0.0.0.0/0"),
			GatewayId:            StringP(natID),
		})
		if err != nil {
			return "", err
		}

		_, err := conn.ec2.AssociateRouteTable(&ec2.AssociateRouteTableInput{
			RouteTableId: out.RouteTable.RouteTableId,
			SubnetId:     StringP(privateSubnetID),
		})

		if err != nil {
			return "", errors.Wrapf(err, "failed to associate route table to subnet %q", privateSubnetID)
		}
	}
	Logger(conn.ctx).Infof("Route added to route table %v", id)

	return id, nil
}

func (conn *cloudConnector) setupSecurityGroups(vpcID string) error {
	var ok bool
	var err error
	if conn.cluster.Status.Cloud.AWS.MasterSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.cluster.Spec.Config.Cloud.AWS.MasterSGName); !ok {
		if err != nil {
			return err
		}
		err = conn.createSecurityGroup(vpcID, conn.cluster.Spec.Config.Cloud.AWS.MasterSGName, "controlplane")
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Master security group %v created", conn.cluster.Spec.Config.Cloud.AWS.MasterSGName)
	}
	if conn.cluster.Status.Cloud.AWS.NodeSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.cluster.Spec.Config.Cloud.AWS.NodeSGName); !ok {
		if err != nil {
			return err
		}
		err = conn.createSecurityGroup(vpcID, conn.cluster.Spec.Config.Cloud.AWS.NodeSGName, "node")
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Node security group %v created", conn.cluster.Spec.Config.Cloud.AWS.NodeSGName)
	}
	if conn.cluster.Status.Cloud.AWS.BastionSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.cluster.Spec.Config.Cloud.AWS.BastionSGName); !ok {
		if err != nil {
			return err
		}
		err = conn.createSecurityGroup(vpcID, conn.cluster.Spec.Config.Cloud.AWS.BastionSGName, "bastion")
		if err != nil {
			return err
		}
		Logger(conn.ctx).Infof("Bastion security group %v created", conn.cluster.Spec.Config.Cloud.AWS.BastionSGName)
	}

	err = conn.detectSecurityGroups(vpcID)
	if err != nil {
		return err
	}

	roles := []string{
		"bastion",
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
			GroupId: StringP(groupID),
		}

		for _, rule := range rules {
			input.IpPermissions = append(input.IpPermissions, rule)
		}

		_, err = conn.ec2.AuthorizeSecurityGroupIngress(input)
		if err != nil {
			return err
		}
	}

	return nil
}

func (conn *cloudConnector) getGroupID(role string) (string, error) {
	switch role {
	case "bastion":
		return conn.cluster.Status.Cloud.AWS.BastionSGId, nil
	case "controlplane":
		return conn.cluster.Status.Cloud.AWS.MasterSGId, nil
	case "node":
		return conn.cluster.Status.Cloud.AWS.NodeSGId, nil
	}
	return "", errors.New("error")
}

// ref: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/c46b0c4de65b97f0043821524f876715d9a65f0c/pkg/cloud/aws/services/ec2/securitygroups.go#L232
func (conn *cloudConnector) getIngressRules(role string) ([]*ec2.IpPermission, error) {
	switch role {
	case "bastion":
		return []*ec2.IpPermission{
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(22),
				ToPort:     Int64P(22),
				IpRanges: []*ec2.IpRange{
					{
						Description: StringP("SSH"),
						CidrIp:      StringP("0.0.0.0/0"),
					},
				},
			},
		}, nil
	case "controlplane":
		return []*ec2.IpPermission{
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(22),
				ToPort:     Int64P(22),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: StringP("SSH"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.BastionSGId),
					},
				},
			},
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(6443),
				ToPort:     Int64P(6443),
				IpRanges: []*ec2.IpRange{
					{
						Description: StringP("Kubernetes API"),
						CidrIp:      StringP("0.0.0.0/0"),
					},
				},
			},
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(2379),
				ToPort:     Int64P(2379),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: StringP("etcd"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.MasterSGId),
					},
				},
			},
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(2380),
				ToPort:     Int64P(2380),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: StringP("etcd peer"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.MasterSGId),
					},
				},
			},
			{
				IpProtocol: StringP("tcp"), //v1alpha1.SecurityGroupProtocolTCP,
				FromPort:   Int64P(179),
				ToPort:     Int64P(179),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: StringP("bgp (calico)"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.MasterSGId),
					},
					{
						Description: StringP("bgp (calico)"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.NodeSGId),
					},
				},
			},
		}, nil

	case "node":
		return []*ec2.IpPermission{
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(22),
				ToPort:     Int64P(22),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: StringP("SSH"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.BastionSGId),
					},
				},
			},
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(30000),
				ToPort:     Int64P(32767),
				IpRanges: []*ec2.IpRange{
					{
						Description: StringP("Node Port Services"),
						CidrIp:      StringP("0.0.0.0/0"),
					},
				}, //[]string{anyIPv4CidrBlock},
			},
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(10250),
				ToPort:     Int64P(10250),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: StringP("Kubelet API"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.MasterSGId),
					},
				},
			},
			{
				IpProtocol: StringP("tcp"),
				FromPort:   Int64P(179),
				ToPort:     Int64P(179),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: StringP("bgp (calico)"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.MasterSGId),
					},
					{
						Description: StringP("bgp (calico)"),
						GroupId:     StringP(conn.cluster.Status.Cloud.AWS.NodeSGId),
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
				Name:   StringP("tag:Name"),
				Values: StringPSlice([]string{conn.namer.BastionName()}),
			},
			{
				Name:   StringP("tag:sigs.k8s.io/cluster-api-provider-aws/role"),
				Values: StringPSlice([]string{"bastion"}),
			},
			{
				Name:   StringP("instance-state-name"),
				Values: StringPSlice([]string{"running"}),
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
				//conn.cluster.Status.Cloud.AWS.BastionID = *instance.InstanceId
				return true, nil
			}
		}
	}

	return false, errors.New("bastion host not found")
}

func (conn *cloudConnector) setupBastion(publicSubnetID string) error {
	sshKeyName := conn.cluster.Spec.Config.Cloud.SSHKeyName
	name := fmt.Sprintf("%s-bastion", conn.cluster.Name)

	userData, err := clusterapi_aws.NewBastion(&clusterapi_aws.BastionInput{})
	if err != nil {
		return err
	}

	input := &ec2.RunInstancesInput{
		InstanceType: StringP("t2.micro"),
		SubnetId:     StringP(publicSubnetID),
		ImageId:      StringP(getBastionAMI(conn.cluster.Spec.Config.Cloud.Region)),
		KeyName:      &sshKeyName,
		MaxCount:     Int64P(1),
		MinCount:     Int64P(1),
		UserData:     StringP(base64.StdEncoding.EncodeToString([]byte(userData))),
		SecurityGroupIds: []*string{
			StringP(conn.cluster.Status.Cloud.AWS.BastionSGId),
		},
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: StringP("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   StringP("Name"),
						Value: StringP(name),
					},
					{
						Key:   StringP("sigs.k8s.io/cluster-api-provider-aws/role"),
						Value: StringP("bastion"),
					},
					{
						Key:   StringP("kubernetes.io/cluster/" + conn.cluster.Name),
						Value: StringP("owned"),
					},
					{
						Key:   StringP("sigs.k8s.io/cluster-api-provider-aws/managed"),
						Value: StringP("true"),
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
	Logger(conn.ctx).Infof("Checking security group %v", groupName)
	r1, err := conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(vpcID),
				},
			},
			{
				Name: StringP("group-name"),
				Values: []*string{
					StringP(groupName),
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

func (conn *cloudConnector) createSecurityGroup(vpcID, groupName string, instanceType string) error {
	Logger(conn.ctx).Infof("Creating security group %v", groupName)
	r2, err := conn.ec2.CreateSecurityGroup(&_ec2.CreateSecurityGroupInput{
		GroupName:   StringP(groupName),
		Description: StringP("kubernetes security group for " + instanceType),
		VpcId:       StringP(vpcID),
	})
	Logger(conn.ctx).Debug("Created security group", r2, err)
	if err != nil {
		return err
	}

	time.Sleep(preTagDelay)

	tags := map[string]string{
		"Name": conn.cluster.Name + "-" + instanceType,
		"kubernetes.io/cluster/" + conn.cluster.Name:   "owned",
		"sigs.k8s.io/cluster-api-provider-aws/managed": "true",
		"sigs.k8s.io/cluster-api-provider-aws/role":    instanceType,
	}

	return conn.addTags(*r2.GroupId, tags)
}

func (conn *cloudConnector) detectSecurityGroups(vpcID string) error {
	var ok bool
	var err error
	if conn.cluster.Status.Cloud.AWS.MasterSGId == "" {
		if conn.cluster.Status.Cloud.AWS.MasterSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.cluster.Spec.Config.Cloud.AWS.MasterSGName); !ok {
			return errors.Errorf("[%s] could not detect Kubernetes master security group.  Make sure you've launched a cluster with appctl", ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Master security group %v with id %v detected", conn.cluster.Spec.Config.Cloud.AWS.MasterSGName, conn.cluster.Status.Cloud.AWS.MasterSGId)

	}
	if conn.cluster.Status.Cloud.AWS.NodeSGId == "" {
		if conn.cluster.Status.Cloud.AWS.NodeSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.cluster.Spec.Config.Cloud.AWS.NodeSGName); !ok {
			return errors.Errorf("[%s] could not detect Kubernetes node security group.  Make sure you've launched a cluster with appctl", ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Node security group %v with id %v detected", conn.cluster.Spec.Config.Cloud.AWS.NodeSGName, conn.cluster.Status.Cloud.AWS.NodeSGId)
	}
	if conn.cluster.Status.Cloud.AWS.BastionSGId == "" {
		if conn.cluster.Status.Cloud.AWS.BastionSGId, ok, err = conn.getSecurityGroupID(vpcID, conn.cluster.Spec.Config.Cloud.AWS.BastionSGName); !ok {
			return errors.Errorf("[%s] could not detect Kubernetes bastion security group.  Make sure you've launched a cluster with appctl", ID(conn.ctx))
		}
		Logger(conn.ctx).Infof("Bastion security group %v with id %v detected", conn.cluster.Spec.Config.Cloud.AWS.BastionSGName, conn.cluster.Status.Cloud.AWS.BastionSGId)
	}
	if err != nil {
		return err
	}
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
				Name:   StringP("tag:sigs.k8s.io/cluster-api-provider-aws/role"),
				Values: StringPSlice([]string{"controlplane"}),
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
	return true, err
}

func (conn *cloudConnector) startMaster(machine *clusterv1.Machine, sku, privateSubnetID string) (*api.NodeInfo, error) {
	sshKeyName := conn.cluster.Spec.Config.Cloud.SSHKeyName

	if err := conn.detectUbuntuImage(); err != nil {
		return nil, err
	}

	script, err := conn.renderStartupScript(machine, "")
	if err != nil {
		return nil, err
	}

	input := &ec2.RunInstancesInput{
		InstanceType: StringP(sku),
		SubnetId:     StringP(privateSubnetID),
		ImageId:      StringP(conn.cluster.Spec.Config.Cloud.InstanceImage),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: StringP(conn.cluster.Spec.Config.Cloud.AWS.IAMProfileMaster),
		},
		KeyName:  &sshKeyName,
		MaxCount: Int64P(1),
		MinCount: Int64P(1),
		UserData: StringP(base64.StdEncoding.EncodeToString([]byte(script))),
		SecurityGroupIds: []*string{
			StringP(conn.cluster.Status.Cloud.AWS.MasterSGId),
		},

		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: StringP("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   StringP("Name"),
						Value: StringP(machine.Name),
					},
					{
						Key:   StringP("sigs.k8s.io/cluster-api-provider-aws/role"),
						Value: StringP(machine.Labels["set"]),
					},
					{
						Key:   StringP("kubernetes.io/cluster/" + conn.cluster.Name),
						Value: StringP("owned"),
					},
					{
						Key:   StringP("sigs.k8s.io/cluster-api-provider-aws/managed"),
						Value: StringP("true"),
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

	Logger(conn.ctx).Info("Waiting for master instance to be ready")
	// We are not able to add an elastic ip, a route or volume to the instance until that instance is in "running" state.
	err = conn.waitForInstanceState(*masterInstance.InstanceId, "running")
	if err != nil {
		return nil, err
	}
	Logger(conn.ctx).Info("Master instance is ready")

	lbinput := &elb.RegisterInstancesWithLoadBalancerInput{
		Instances:        []*elb.Instance{{InstanceId: masterInstance.InstanceId}},
		LoadBalancerName: StringP(fmt.Sprintf("%s-apiserver", conn.cluster.Name)),
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
	node := api.NodeInfo{
		Name:       *r.Reservations[0].Instances[0].PrivateDnsName,
		ExternalID: *masterInstance.InstanceId,
	}

	return &node, nil
}

func (conn *cloudConnector) waitForInstanceState(instanceID, state string) error {
	for {
		r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
			InstanceIds: []*string{StringP(instanceID)},
		})
		if err != nil {
			return err
		}
		curState := *r1.Reservations[0].Instances[0].State.Name
		if curState == state {
			break
		}
		Logger(conn.ctx).Infof("Waiting for instance %v to be %v (currently %v)", instanceID, state, curState)
		Logger(conn.ctx).Infof("Sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (conn *cloudConnector) allocateElasticIP() (string, string, error) {
	r1, err := conn.ec2.AllocateAddress(&_ec2.AllocateAddressInput{
		Domain: StringP("vpc"),
	})
	Logger(conn.ctx).Debug("Allocated elastic IP", r1, err)
	if err != nil {
		return "", "", err
	}
	Logger(conn.ctx).Infof("Elastic IP %v allocated", *r1.PublicIp)
	time.Sleep(5 * time.Second)

	tags := map[string]string{
		"Name": conn.cluster.Name + "-eip-apiserver",
		"kubernetes.io/cluster/" + conn.cluster.Name:   "owned",
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
				Name: StringP("tag:Name"),
				Values: []*string{
					StringP(conn.cluster.Name + "-eip-apiserver"),
				},
			},
			{
				Name: StringP("tag-key"),
				Values: []*string{
					StringP("kubernetes.io/cluster/" + conn.cluster.Name),
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

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (done bool, err error) {
		fmt.Println("waiting for elastic ip to be released")
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
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (done bool, err error) {
		r, err := conn.ec2.DescribeSecurityGroups(&_ec2.DescribeSecurityGroupsInput{
			Filters: []*_ec2.Filter{
				{
					Name: StringP("vpc-id"),
					Values: []*string{
						StringP(vpcID),
					},
				},
				{
					Name: StringP("tag-key"),
					Values: []*string{
						StringP("kubernetes.io/cluster/" + conn.cluster.Name),
					},
				},
			},
		})
		if err != nil {
			return false, nil
		}

		for _, sg := range r.SecurityGroups {
			if len(sg.IpPermissions) > 0 {
				_, err := conn.ec2.RevokeSecurityGroupIngress(&_ec2.RevokeSecurityGroupIngressInput{
					GroupId:       sg.GroupId,
					IpPermissions: sg.IpPermissions,
				})
				if err != nil {
					log.Infof(err.Error())
					return false, nil
				}
			}

			if len(sg.IpPermissionsEgress) > 0 {
				_, err := conn.ec2.RevokeSecurityGroupEgress(&_ec2.RevokeSecurityGroupEgressInput{
					GroupId:       sg.GroupId,
					IpPermissions: sg.IpPermissionsEgress,
				})
				if err != nil {
					log.Infof(err.Error())
					return false, nil
				}
			}
		}

		for _, sg := range r.SecurityGroups {
			_, err := conn.ec2.DeleteSecurityGroup(&_ec2.DeleteSecurityGroupInput{
				GroupId: sg.GroupId,
			})
			if err != nil {
				log.Infof(err.Error())
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
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(vpcID),
				},
			},
			{
				Name: StringP("tag-key"),
				Values: []*string{
					StringP("kubernetes.io/cluster/" + conn.cluster.Name),
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

func (conn *cloudConnector) deleteInternetGateway(vpcID string) error {
	r1, err := conn.ec2.DescribeInternetGateways(&_ec2.DescribeInternetGatewaysInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("attachment.vpc-id"),
				Values: []*string{
					StringP(vpcID),
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
			VpcId:             StringP(vpcID),
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

func (conn *cloudConnector) deleteRouteTable(vpcID string) error {
	r1, err := conn.ec2.DescribeRouteTables(&_ec2.DescribeRouteTablesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("vpc-id"),
				Values: []*string{
					StringP(vpcID),
				},
			},
			{
				Name: StringP("tag-key"),
				Values: []*string{
					StringP("kubernetes.io/cluster/" + conn.cluster.Name),
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
	Logger(conn.ctx).Infof("Route tables for cluster %v are deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteNatGateway(natID string) error {
	if _, err := conn.ec2.DeleteNatGateway(&ec2.DeleteNatGatewayInput{
		NatGatewayId: StringP(natID),
	}); err != nil {
		return err
	}

	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		fmt.Println("waiting for nat to be deleted")
		out, err := conn.ec2.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []*string{
				StringP(natID),
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
		VpcId: StringP(vpcID),
	})

	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("VPC for cluster %v is deleted", conn.cluster.Name)
	return nil
}

func (conn *cloudConnector) deleteSSHKey() error {
	var err error
	_, err = conn.ec2.DeleteKeyPair(&_ec2.DeleteKeyPairInput{
		KeyName: StringP(conn.cluster.Spec.Config.Cloud.SSHKeyName),
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("SSH key for cluster %v is deleted", conn.cluster.Name)

	return err
}

func (conn *cloudConnector) deleteInstance(role string) error {
	log.Infof("deleting instance %s", role)
	r1, err := conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
		Filters: []*_ec2.Filter{
			{
				Name: StringP("tag:sigs.k8s.io/cluster-api-provider-aws/role"),
				Values: []*string{
					StringP(role),
				},
			},
			{
				Name: StringP("tag-key"),
				Values: []*string{
					StringP(fmt.Sprintf("kubernetes.io/cluster/%s", conn.cluster.Name)),
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
	Logger(conn.ctx).Infof("Terminating %v instance for cluster %v", role, conn.cluster.Name)
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
	Logger(conn.ctx).Infof("%v instance for cluster %v is terminated", role, conn.cluster.Name)
	return err
}
