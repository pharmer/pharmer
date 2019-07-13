package eks

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/appscode/go/types"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_eks "github.com/aws/aws-sdk-go/service/eks"
	_iam "github.com/aws/aws-sdk-go/service/iam"
	_sts "github.com/aws/aws-sdk-go/service/sts"
	"gomodules.xyz/version"
	"k8s.io/apimachinery/pkg/util/wait" //"pharmer.dev/pharmer/cloud/providers/eks/assets"
	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cloud"
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
	log := cm.Logger
	cluster := cm.Cluster
	cred, err := cm.StoreProvider.Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		log.Error(err, "failed to get credential from store")
		return nil, err
	}
	typed := credential.AWS{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		log.Error(err, "credential is invalid", "credential-name", cluster.Spec.Config.CredentialName)
		return nil, err
	}

	config := &_aws.Config{
		Region:      &cluster.Spec.Config.Cloud.Region,
		Credentials: credentials.NewStaticCredentials(typed.AccessKeyID(), typed.SecretAccessKey(), ""),
	}
	sess, err := session.NewSession(config)
	if err != nil {
		log.Error(err, "failed to create new session")
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
	log := conn.Logger

	attempt := 0
	params := &cloudformation.DescribeStacksInput{
		StackName: types.StringP(name),
	}
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		resp, err := conn.cfn.DescribeStacks(params)
		if err != nil {
			log.Error(err, "error describing stack")
			return false, nil
		}
		status := *resp.Stacks[0].StackStatus
		log.Info("waiting for stack operation", "attempt", attempt, "operation", name, "status", status)
		return status == expectedStatus, nil
	})
}

func (conn *cloudConnector) WaitForControlPlaneOperation(name string) error {
	log := conn.Logger

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

		log.Info("waiting for control plane operation", "attempt", attempt, "operation", name, "status", status)

		return status == _eks.ClusterStatusActive, nil
	})
}

func (conn *cloudConnector) ensureStackServiceRole() error {
	log := conn.Logger

	found := conn.isStackExists(conn.namer.GetStackServiceRole())
	serviceRoleName := conn.namer.GetStackServiceRole()

	if !found {
		if err := conn.createStack(serviceRoleName, ServiceRoleURL, nil, true); err != nil {
			log.Error(err, "failed to create service role")
			return err
		}
	}

	serviceRole, err := conn.getStack(serviceRoleName)
	if err != nil {
		log.Error(err, "failed to get service role", "service-role-name", serviceRoleName)
		return err
	}

	roleArn := conn.getOutput(serviceRole, "RoleArn")
	if roleArn == nil {
		return fmt.Errorf("RoleArn is nil")
	}

	conn.Cluster.Status.Cloud.EKS.RoleArn = types.String(roleArn)

	return nil
}

func (conn *cloudConnector) ensureClusterVPC() error {
	log := conn.Logger

	found := conn.isStackExists(conn.namer.GetClusterVPC())
	vpcName := conn.namer.GetClusterVPC()
	if !found {
		if err := conn.createStack(vpcName, EKSVPCUrl, nil, false); err != nil {
			log.Error(err, "failed to create cluster")
			return err
		}
	}

	vpc, err := conn.getStack(vpcName)
	if err != nil {
		log.Error(err, "failed to get vpc", "vpc-name", vpcName)
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

	vpcID := conn.getOutput(vpc, "VpcId")
	if vpcID == nil {
		return fmt.Errorf("VpcID is nil")
	}
	conn.Cluster.Status.Cloud.EKS.VpcID = types.String(vpcID)
	return nil
}

func (conn *cloudConnector) createControlPlane() error {
	log := conn.Logger
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
	log.V(4).Info("creating cluster", "cluster-params", params)
	if err != nil {
		log.Error(err, "failed to create cluser")
		return err
	}
	return conn.WaitForControlPlaneOperation(conn.Cluster.Name)
}

func (conn *cloudConnector) deleteControlPlane() error {
	log := conn.Logger
	params := &_eks.DeleteClusterInput{
		Name: types.StringP(conn.Cluster.Name),
	}
	_, err := conn.eks.DeleteCluster(params)
	log.V(4).Info("deleting cluster", "cluster-params", params)
	if err != nil {
		log.Error(err, "failed to delete cluster")
		return err
	}
	return nil
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
	log := conn.Logger.WithValues("stack-name", name)
	params := &cloudformation.DescribeStacksInput{
		StackName: types.StringP(name),
	}
	resp, err := conn.cfn.DescribeStacks(params)
	if err != nil {
		log.Error(err, "failed to describe stacks")
		return nil, err
	}
	if len(resp.Stacks) == 1 {
		return resp.Stacks[0], nil
	}
	return nil, fmt.Errorf("stack %v not exists", name)
}

func (conn *cloudConnector) isStackExists(name string) bool {
	log := conn.Logger.WithValues("stack-name", name)

	log.Info("Checking if stack exists")
	params := &cloudformation.DescribeStacksInput{
		StackName: types.StringP(name),
	}
	resp, err := conn.cfn.DescribeStacks(params)
	if err != nil {
		log.Error(err, "failed to describe stacks")
		return false
	}
	if len(resp.Stacks) > 0 {
		return true
	}
	return false
}

func (conn *cloudConnector) isControlPlaneExists(name string) bool {
	log := conn.Logger.WithValues("control-plane-name", name)
	log.Info("Checking for control plane exists")
	params := &_eks.DescribeClusterInput{
		Name: types.StringP(name),
	}
	resp, err := conn.eks.DescribeCluster(params)
	if err != nil {
		log.Error(err, "failed to describe eks cluster")
		return false
	}
	if resp.Cluster != nil {
		return true
	}
	return false
}

func (conn *cloudConnector) createStack(name, url string, params map[string]string, withIAM bool) error {
	log := conn.Logger.WithValues("stack-name", name)
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
		log.Error(err, "failed to create stack")
		return err
	}
	return conn.WaitForStackOperation(name, cloudformation.StackStatusCreateComplete)
}

func (conn *cloudConnector) deleteStack(name string) error {
	log := conn.Logger.WithValues("stack-name", name)

	log.Info("Deleting stack")
	params := &cloudformation.DeleteStackInput{
		StackName: types.StringP(name),
	}
	_, err := conn.cfn.DeleteStack(params)
	if err != nil {
		log.Error(err, "failed to delete stack")
	}

	return nil
}

func (conn *cloudConnector) updateStack(name string, params map[string]string, withIAM bool) error {
	log := conn.Logger

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
		log.Error(err, "failed to update stack")
		return nil
	}

	return conn.WaitForStackOperation(name, cloudformation.StackStatusUpdateComplete)
}

func (conn *cloudConnector) getPublicKey() (bool, error) {
	log := conn.Logger

	resp, err := conn.ec2.DescribeKeyPairs(&_ec2.DescribeKeyPairsInput{
		KeyNames: types.StringPSlice([]string{conn.Cluster.Spec.Config.Cloud.SSHKeyName}),
	})
	if err != nil {
		log.Error(err, "failed to describe ec2 key pair")
		return false, err
	}
	if len(resp.KeyPairs) > 0 {
		return true, nil
	}
	return false, nil
}

func (conn *cloudConnector) importPublicKey() error {
	log := conn.Logger

	resp, err := conn.ec2.ImportKeyPair(&_ec2.ImportKeyPairInput{
		KeyName:           types.StringP(conn.Cluster.Spec.Config.Cloud.SSHKeyName),
		PublicKeyMaterial: conn.Certs.SSHKey.PublicKey,
	})
	log.V(2).Info("Import SSH key", "response", resp)
	if err != nil {
		log.Error(err, "Error importing public key")
		return err
	}
	log.Info("SSH key with (AWS) fingerprint %v imported", "fingerprint", conn.Certs.SSHKey.AwsFingerprint)

	return nil
}

func (conn *cloudConnector) deleteSSHKey() error {
	log := conn.Logger

	var err error
	_, err = conn.ec2.DeleteKeyPair(&_ec2.DeleteKeyPairInput{
		KeyName: types.StringP(conn.Cluster.Spec.Config.Cloud.SSHKeyName),
	})
	if err != nil {
		log.Error(err, "failed to delete ec2 key pair")
		return err
	}
	log.Info("SSH key deleted")

	return err
}

// ---------------------------------------------------------------------------------------------------------------------

func (conn *cloudConnector) getAuthenticationToken() (string, error) {
	request, _ := conn.sts.GetCallerIdentityRequest(&_sts.GetCallerIdentityInput{})
	request.HTTPRequest.Header.Add(clusterIDHeader, conn.Cluster.Name)
	presignedURLString, err := request.Presign(60 * time.Second)
	if err != nil {
		conn.Logger.Error(err, "failed to sign url")
		return "", err
	}
	token := v1Prefix + base64.RawURLEncoding.EncodeToString([]byte(presignedURLString))
	return token, nil
}
