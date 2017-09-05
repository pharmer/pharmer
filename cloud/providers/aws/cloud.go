package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/credential"
	_aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
	_elb "github.com/aws/aws-sdk-go/service/elb"
	_iam "github.com/aws/aws-sdk-go/service/iam"
	_s3 "github.com/aws/aws-sdk-go/service/s3"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster

	ec2       *_ec2.EC2
	elb       *_elb.ELB
	iam       *_iam.IAM
	autoscale *autoscaling.AutoScaling
	s3        *_s3.S3
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := cloud.Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.AWS{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.New().WithMessagef("Credential %s is invalid. Reason: %v", cluster.Spec.CredentialName, err)
	}

	config := &_aws.Config{
		Region:      &cluster.Spec.Region,
		Credentials: credentials.NewStaticCredentials(typed.AccessKeyID(), typed.SecretAccessKey(), ""),
	}
	conn := cloudConnector{
		ctx:       ctx,
		cluster:   cluster,
		ec2:       _ec2.New(session.New(config)),
		elb:       _elb.New(session.New(config)),
		iam:       _iam.New(session.New(config)),
		autoscale: autoscaling.New(session.New(config)),
		s3:        _s3.New(session.New(config)),
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		return nil, fmt.Errorf("Credential %s does not have necessary authorization. Reason: %s.", cluster.Spec.CredentialName, msg)
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

// https://github.com/kubernetes/kubernetes/blob/master/cluster/aws/jessie/util.sh#L28
// Based on https://github.com/kubernetes/kube-deploy/tree/master/imagebuilder
func (conn *cloudConnector) detectJessieImage() error {
	conn.cluster.Spec.OS = "debian"
	r1, err := conn.ec2.DescribeImages(&_ec2.DescribeImagesInput{
		Owners: []*string{StringP("282335181503")},
		Filters: []*_ec2.Filter{
			{
				Name: StringP("name"),
				Values: []*string{
					StringP("k8s-1.3-debian-jessie-amd64-hvm-ebs-2016-06-18"),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.cluster.Spec.InstanceImage = *r1.Images[0].ImageId
	conn.cluster.Spec.RootDeviceName = *r1.Images[0].RootDeviceName
	cloud.Logger(conn.ctx).Infof("Debain image with %v for %v detected", conn.cluster.Spec.InstanceImage, conn.cluster.Spec.RootDeviceName)
	return nil
}

func (conn *cloudConnector) detectUbuntuImage() error {
	conn.cluster.Spec.OS = "ubuntu"
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
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.cluster.Spec.InstanceImage = *r1.Images[0].ImageId
	conn.cluster.Spec.RootDeviceName = *r1.Images[0].RootDeviceName
	cloud.Logger(conn.ctx).Infof("Ubuntu image with %v for %v detected", conn.cluster.Spec.InstanceImage, conn.cluster.Spec.RootDeviceName)
	return nil
}
