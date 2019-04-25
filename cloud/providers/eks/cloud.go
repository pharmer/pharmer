package eks

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	version "github.com/appscode/go-version"
	"github.com/appscode/go/log"
	. "github.com/appscode/go/types"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_eks "github.com/aws/aws-sdk-go/service/eks"
	_iam "github.com/aws/aws-sdk-go/service/iam"
	_sts "github.com/aws/aws-sdk-go/service/sts"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait" //"github.com/pharmer/pharmer/cloud/providers/eks/assets"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	namer   namer

	ec2 *_ec2.EC2
	iam *_iam.IAM
	eks *_eks.EKS
	sts *_sts.STS
	//stscreds *_stscreds.AssumeRoler
	cfn *cloudformation.CloudFormation
}

func NewConnector(ctx context.Context, cluster *api.Cluster, owner string) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.AWS{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Errorf("credential %s is invalid. Reason: %v", cluster.Spec.Config.CredentialName, err)
	}

	config := &_aws.Config{
		Region:      &cluster.Spec.Config.Cloud.Region,
		Credentials: credentials.NewStaticCredentials(typed.AccessKeyID(), typed.SecretAccessKey(), ""),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}
	conn := cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		eks:     _eks.New(sess),
		ec2:     _ec2.New(sess),
		iam:     _iam.New(sess),
		sts:     _sts.New(sess),
		cfn:     cloudformation.New(sess),
	}
	//if ok, msg := conn.IsUnauthorized(); !ok {
	//	return nil, errors.Errorf("credential %s does not have necessary authorization. Reason: %s", cluster.Spec.CredentialName, msg)
	//}
	return &conn, nil
}

func (conn *cloudConnector) DetectInstanceImage() (string, error) {
	v10, err := version.NewVersion("1.10")
	if err != nil {
		return "", err
	}
	cv, err := version.NewVersion(conn.cluster.Spec.Config.KubernetesVersion)
	if err != nil {
		return "", err
	}
	var regionalAMIS map[string]string
	if cv.Equal(v10) {
		regionalAMIS = map[string]string{
			// https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html
			"us-west-2": "ami-0e36fae01a5fa0d76",
			"us-east-1": "ami-0de0b13514617a168",
		}
	} else {
		regionalAMIS = map[string]string{
			"us-west-2": "ami-081099ec932b99961",
			"us-east-1": "ami-0c5b63ec54dd3fc38",
		}
	}

	return regionalAMIS[conn.cluster.Spec.Config.Cloud.Region], nil
}

func (conn *cloudConnector) WaitForStackOperation(name string, expectedStatus string) error {
	attempt := 0
	params := &cloudformation.DescribeStacksInput{
		StackName: StringP(name),
	}
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		resp, err := conn.cfn.DescribeStacks(params)
		if err != nil {
			log.Info(err)
		}
		if err != nil {
			return false, nil
		}
		status := *resp.Stacks[0].StackStatus
		Logger(conn.ctx).Infof("Attempt %v: operation `%s` is in status `%s`", attempt, name, status)
		return status == expectedStatus, nil
	})
}

func (conn *cloudConnector) WaitForControlPlaneOperation(name string) error {
	attempt := 0
	params := &_eks.DescribeClusterInput{
		Name: StringP(name),
	}
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		resp, err := conn.eks.DescribeCluster(params)
		if err != nil {
			return false, nil
		}
		status := *resp.Cluster.Status

		Logger(conn.ctx).Infof("Attempt %v: operation `%s` is in status `%s`", attempt, name, status)
		return status == _eks.ClusterStatusActive, nil
	})
}

func (conn *cloudConnector) createStackServiceRole() error {
	/*data, err := Asset("amazon-eks-service-role.yaml")
	if err != nil {
		return err
	}*/
	serviceRoleName := conn.namer.GetStackServiceRole()
	if err := conn.createStack(serviceRoleName, ServiceRoleUrl, nil, true); err != nil {
		return err
	}
	serviceRole, err := conn.getStack(serviceRoleName)
	if err != nil {
		return err
	}
	roleArn := conn.getOutput(serviceRole, "RoleArn")
	if roleArn == nil {
		return fmt.Errorf("RoleArn is nil")
	}
	conn.cluster.Status.Cloud.EKS.RoleArn = String(roleArn)
	return nil
}

func (conn *cloudConnector) createClusterVPC() error {
	//https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-06-05/amazon-eks-vpc-sample.yaml
	/*data, err := Asset("amazon-eks-vpc-sample.yaml")
	if err != nil {
		return err
	}*/
	vpcName := conn.namer.GetClusterVPC()
	if err := conn.createStack(vpcName, EKSVPCUrl, nil, false); err != nil {
		return err
	}
	vpc, err := conn.getStack(vpcName)
	if err != nil {
		return err
	}

	securityGroup := conn.getOutput(vpc, "SecurityGroups")
	if securityGroup == nil {
		return fmt.Errorf("SecurityGroups is nil")
	}
	conn.cluster.Status.Cloud.EKS.SecurityGroup = String(securityGroup)

	subnetIds := conn.getOutput(vpc, "SubnetIds")
	if subnetIds == nil {
		return fmt.Errorf("SubnetIds is nil")
	}
	conn.cluster.Status.Cloud.EKS.SubnetId = String(subnetIds)

	vpcId := conn.getOutput(vpc, "VpcId")
	if vpcId == nil {
		return fmt.Errorf("VpcId is nil")
	}
	conn.cluster.Status.Cloud.EKS.VpcId = String(vpcId)
	return nil
}

func (conn *cloudConnector) createStackNodeGroup() {

}
func (conn *cloudConnector) createControlPlane() error {
	params := &_eks.CreateClusterInput{
		Name:    StringP(conn.cluster.Name),
		RoleArn: StringP(conn.cluster.Status.Cloud.EKS.RoleArn),
		ResourcesVpcConfig: &_eks.VpcConfigRequest{
			SubnetIds:        StringPSlice(strings.Split(conn.cluster.Status.Cloud.EKS.SubnetId, ",")),
			SecurityGroupIds: StringPSlice([]string{conn.cluster.Status.Cloud.EKS.SecurityGroup}),
		},
		Version: StringP(conn.cluster.Spec.Config.KubernetesVersion),
	}
	_, err := conn.eks.CreateCluster(params)
	if err != nil {
		return err
	}
	return conn.WaitForControlPlaneOperation(conn.cluster.Name)
}

func (conn *cloudConnector) deleteControlPlane() error {
	params := &_eks.DeleteClusterInput{
		Name: StringP(conn.cluster.Name),
	}
	_, err := conn.eks.DeleteCluster(params)
	return err
}

func (conn *cloudConnector) getOutput(stack *cloudformation.Stack, key string) *string {
	for _, x := range stack.Outputs {
		if *x.OutputKey == key {
			return x.OutputValue
		}
	}
	return nil
}

func (conn *cloudConnector) getStack(name string) (*cloudformation.Stack, error) {
	params := &cloudformation.DescribeStacksInput{
		StackName: StringP(name),
	}
	resp, err := conn.cfn.DescribeStacks(params)
	if err != nil {
		return nil, err
	}
	if len(resp.Stacks) == 1 {
		return resp.Stacks[0], nil
	}
	return nil, fmt.Errorf("stack %v not exists", name)
}

func (conn *cloudConnector) isStackExists(name string) (bool, error) {
	Logger(conn.ctx).Infof("Checking if %v exists...", name)
	params := &cloudformation.DescribeStacksInput{
		StackName: StringP(name),
	}
	resp, err := conn.cfn.DescribeStacks(params)
	if err != nil {
		return false, nil
	}
	if len(resp.Stacks) > 0 {
		return true, nil
	}
	return false, nil //fmt.Errorf("stack %v not exists", name)
}

func (conn *cloudConnector) isControlPlaneExists(name string) (bool, error) {
	Logger(conn.ctx).Infof("Checking for control plane %v exists...", name)
	params := &_eks.DescribeClusterInput{
		Name: StringP(name),
	}
	resp, err := conn.eks.DescribeCluster(params)
	if err != nil {
		return false, nil
	}
	if resp.Cluster != nil {
		return true, nil
	}
	return false, nil
}

func (conn *cloudConnector) createStack(name, url string, params map[string]string, withIAM bool) error {
	cfn := &cloudformation.CreateStackInput{}
	cfn.SetStackName(name)
	cfn.SetTags([]*cloudformation.Tag{
		{
			Key:   StringP("KubernetesCluster"),
			Value: StringP(conn.cluster.Name),
		},
	})
	cfn.SetTemplateURL(url)
	if withIAM {
		cfn.SetCapabilities(StringPSlice([]string{cloudformation.CapabilityCapabilityIam}))
	}

	for k, v := range params {
		p := &cloudformation.Parameter{
			ParameterKey:   StringP(k),
			ParameterValue: StringP(v),
		}
		cfn.Parameters = append(cfn.Parameters, p)
	}
	_, err := conn.cfn.CreateStack(cfn)
	if err != nil {
		return err
	}
	return conn.WaitForStackOperation(name, cloudformation.StackStatusCreateComplete)
}

func (conn *cloudConnector) deleteStack(name string) error {
	Logger(conn.ctx).Infoln("Deleting stack ", name)
	params := &cloudformation.DeleteStackInput{
		StackName: StringP(name),
	}
	_, err := conn.cfn.DeleteStack(params)
	return err
}

func (conn *cloudConnector) updateStack(name string, params map[string]string, withIAM bool, arn *string) error {
	cfn := &cloudformation.UpdateStackInput{}
	cfn.SetStackName(name)
	cfn.SetTags([]*cloudformation.Tag{
		{
			Key:   StringP("KubernetesCluster"),
			Value: StringP(conn.cluster.Name),
		},
	})
	cfn.SetUsePreviousTemplate(true)
	if withIAM {
		cfn.SetCapabilities(StringPSlice([]string{cloudformation.CapabilityCapabilityIam}))
	}
	for k, v := range params {
		p := &cloudformation.Parameter{
			ParameterKey:   StringP(k),
			ParameterValue: StringP(v),
		}
		cfn.Parameters = append(cfn.Parameters, p)
	}
	//cfn.RoleARN = arn

	_, err := conn.cfn.UpdateStack(cfn)
	if err != nil {
		return err
	}
	return conn.WaitForStackOperation(name, cloudformation.StackStatusUpdateComplete)
}

// ---------------------------------------------------------------------------------------------------------------------

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

func (conn *cloudConnector) deleteSSHKey() error {
	var err error
	_, err = conn.ec2.DeleteKeyPair(&_ec2.DeleteKeyPairInput{
		KeyName: StringP(conn.cluster.Spec.Config.Cloud.SSHKeyName),
	})
	if err != nil {
		return err
	}
	Logger(conn.ctx).Infof("SSH key for cluster %v is deleted", conn.cluster.Name)
	//updates := &storage.SSHKey{IsDeleted: 1}
	//cond := &storage.SSHKey{PHID: cluster.Spec.ctx.SSHKeyPHID}
	// _, err = cluster.Spec.Store(ctx).Engine.Update(updates, cond)

	return err
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getAuthenticationToken() (string, error) {
	request, _ := conn.sts.GetCallerIdentityRequest(&_sts.GetCallerIdentityInput{})
	request.HTTPRequest.Header.Add(clusterIDHeader, conn.cluster.Name)
	// sign the request
	presignedURLString, err := request.Presign(60 * time.Second)
	if err != nil {
		return "", err
	}
	token := v1Prefix + base64.RawURLEncoding.EncodeToString([]byte(presignedURLString))
	return token, nil
}
