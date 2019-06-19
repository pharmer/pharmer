package eks

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
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
	"github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"gomodules.xyz/version"
	"k8s.io/apimachinery/pkg/util/wait" //"github.com/pharmer/pharmer/cloud/providers/eks/assets"
)

type cloudConnector struct {
	*cloud.Scope
	namer namer

	ec2 *_ec2.EC2
	iam *_iam.IAM
	eks *_eks.EKS
	sts *_sts.STS
	cfn *cloudformation.CloudFormation
}

func newconnector(cm *ClusterManager) (*cloudConnector, error) {
	cluster := cm.Cluster
	cred, err := cm.StoreProvider.Credentials().Get(cluster.Spec.Config.CredentialName)
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
		Scope: cm.Scope,
		namer: namer{
			cluster: cm.Cluster,
		},
		eks: _eks.New(sess),
		ec2: _ec2.New(sess),
		iam: _iam.New(sess),
		sts: _sts.New(sess),
		cfn: cloudformation.New(sess),
	}

	return &conn, nil
}

func (conn *cloudConnector) DetectInstanceImage() (string, error) {
	v10, err := version.NewVersion("1.10")
	if err != nil {
		return "", err
	}
	cv, err := version.NewVersion(conn.Cluster.Spec.Config.KubernetesVersion)
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

	return regionalAMIS[conn.Cluster.Spec.Config.Cloud.Region], nil
}

func (conn *cloudConnector) WaitForStackOperation(name string, expectedStatus string) error {
	attempt := 0
	params := &cloudformation.DescribeStacksInput{
		StackName: types.StringP(name),
	}
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		resp, err := conn.cfn.DescribeStacks(params)
		if err != nil {
			log.Info(err)
			return false, nil
		}
		status := *resp.Stacks[0].StackStatus
		log.Infof("Attempt %v: operation `%s` is in status `%s`", attempt, name, status)
		return status == expectedStatus, nil
	})
}

func (conn *cloudConnector) WaitForControlPlaneOperation(name string) error {
	attempt := 0
	params := &_eks.DescribeClusterInput{
		Name: types.StringP(name),
	}
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		resp, err := conn.eks.DescribeCluster(params)
		if err != nil {
			return false, nil
		}
		status := *resp.Cluster.Status

		log.Infof("Attempt %v: operation `%s` is in status `%s`", attempt, name, status)
		return status == _eks.ClusterStatusActive, nil
	})
}

func (conn *cloudConnector) createStackServiceRole() error {
	serviceRoleName := conn.namer.GetStackServiceRole()
	if err := conn.createStack(serviceRoleName, ServiceRoleURL, nil, true); err != nil {
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
	conn.Cluster.Status.Cloud.EKS.RoleArn = types.String(roleArn)
	return nil
}

func (conn *cloudConnector) createClusterVPC() error {
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
	conn.Cluster.Status.Cloud.EKS.SecurityGroup = types.String(securityGroup)

	subnetIds := conn.getOutput(vpc, "SubnetIds")
	if subnetIds == nil {
		return fmt.Errorf("SubnetIds is nil")
	}
	conn.Cluster.Status.Cloud.EKS.SubnetID = types.String(subnetIds)

	vpcID := conn.getOutput(vpc, "VpcID")
	if vpcID == nil {
		return fmt.Errorf("VpcID is nil")
	}
	conn.Cluster.Status.Cloud.EKS.VpcID = types.String(vpcID)
	return nil
}

func (conn *cloudConnector) createControlPlane() error {
	params := &_eks.CreateClusterInput{
		Name:    types.StringP(conn.Cluster.Name),
		RoleArn: types.StringP(conn.Cluster.Status.Cloud.EKS.RoleArn),
		ResourcesVpcConfig: &_eks.VpcConfigRequest{
			SubnetIds:        types.StringPSlice(strings.Split(conn.Cluster.Status.Cloud.EKS.SubnetID, ",")),
			SecurityGroupIds: types.StringPSlice([]string{conn.Cluster.Status.Cloud.EKS.SecurityGroup}),
		},
		Version: types.StringP(conn.Cluster.Spec.Config.KubernetesVersion),
	}
	_, err := conn.eks.CreateCluster(params)
	if err != nil {
		return err
	}
	return conn.WaitForControlPlaneOperation(conn.Cluster.Name)
}

func (conn *cloudConnector) deleteControlPlane() error {
	params := &_eks.DeleteClusterInput{
		Name: types.StringP(conn.Cluster.Name),
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
		StackName: types.StringP(name),
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

func (conn *cloudConnector) isStackExists(name string) bool {
	log.Infof("Checking if %v exists...", name)
	params := &cloudformation.DescribeStacksInput{
		StackName: types.StringP(name),
	}
	resp, err := conn.cfn.DescribeStacks(params)
	if err != nil {
		return false
	}
	if len(resp.Stacks) > 0 {
		return true
	}
	return false
}

func (conn *cloudConnector) isControlPlaneExists(name string) bool {
	log.Infof("Checking for control plane %v exists...", name)
	params := &_eks.DescribeClusterInput{
		Name: types.StringP(name),
	}
	resp, err := conn.eks.DescribeCluster(params)
	if err != nil {
		return false
	}
	if resp.Cluster != nil {
		return true
	}
	return false
}

func (conn *cloudConnector) createStack(name, url string, params map[string]string, withIAM bool) error {
	cfn := &cloudformation.CreateStackInput{}
	cfn.SetStackName(name)
	cfn.SetTags([]*cloudformation.Tag{
		{
			Key:   types.StringP("KubernetesCluster"),
			Value: types.StringP(conn.Cluster.Name),
		},
	})
	cfn.SetTemplateURL(url)
	if withIAM {
		cfn.SetCapabilities(types.StringPSlice([]string{cloudformation.CapabilityCapabilityIam}))
	}

	for k, v := range params {
		p := &cloudformation.Parameter{
			ParameterKey:   types.StringP(k),
			ParameterValue: types.StringP(v),
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
	log.Infoln("Deleting stack ", name)
	params := &cloudformation.DeleteStackInput{
		StackName: types.StringP(name),
	}
	_, err := conn.cfn.DeleteStack(params)
	return err
}

func (conn *cloudConnector) updateStack(name string, params map[string]string, withIAM bool) error {
	cfn := &cloudformation.UpdateStackInput{}
	cfn.SetStackName(name)
	cfn.SetTags([]*cloudformation.Tag{
		{
			Key:   types.StringP("KubernetesCluster"),
			Value: types.StringP(conn.Cluster.Name),
		},
	})
	cfn.SetUsePreviousTemplate(true)
	if withIAM {
		cfn.SetCapabilities(types.StringPSlice([]string{cloudformation.CapabilityCapabilityIam}))
	}
	for k, v := range params {
		p := &cloudformation.Parameter{
			ParameterKey:   types.StringP(k),
			ParameterValue: types.StringP(v),
		}
		cfn.Parameters = append(cfn.Parameters, p)
	}

	_, err := conn.cfn.UpdateStack(cfn)
	if err != nil {
		return nil
	}

	return conn.WaitForStackOperation(name, cloudformation.StackStatusUpdateComplete)
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
	if err != nil {
		log.Info("Error importing public key", resp, err)
		return err
	}
	log.Infof("SSH key with (AWS) fingerprint %v imported", conn.Certs.SSHKey.AwsFingerprint)

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

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getAuthenticationToken() (string, error) {
	request, _ := conn.sts.GetCallerIdentityRequest(&_sts.GetCallerIdentityInput{})
	request.HTTPRequest.Header.Add(clusterIDHeader, conn.Cluster.Name)
	presignedURLString, err := request.Presign(60 * time.Second)
	if err != nil {
		return "", err
	}
	token := v1Prefix + base64.RawURLEncoding.EncodeToString([]byte(presignedURLString))
	return token, nil
}
