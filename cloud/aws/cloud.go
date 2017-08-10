package aws

import (
	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/contexts"
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
	ctx *contexts.ClusterContext

	ec2       *_ec2.EC2
	elb       *_elb.ELB
	iam       *_iam.IAM
	autoscale *autoscaling.AutoScaling
	s3        *_s3.S3
}

func NewConnector(ctx *contexts.ClusterContext) (*cloudConnector, error) {
	id := ctx.CloudCredential[credential.AWSAccessKeyID]
	secret := ctx.CloudCredential[credential.AWSSecretAccessKey]
	config := &_aws.Config{
		Region:      &ctx.Region,
		Credentials: credentials.NewStaticCredentials(id, secret, ""),
	}

	return &cloudConnector{
		ctx:       ctx,
		ec2:       _ec2.New(session.New(config)),
		elb:       _elb.New(session.New(config)),
		iam:       _iam.New(session.New(config)),
		autoscale: autoscaling.New(session.New(config)),
		s3:        _s3.New(session.New(config)),
	}, nil
}

// https://github.com/kubernetes/kubernetes/blob/master/cluster/aws/jessie/util.sh#L28
// Based on https://github.com/kubernetes/kube-deploy/tree/master/imagebuilder
func (conn *cloudConnector) detectJessieImage() error {
	conn.ctx.OS = "debian"
	r1, err := conn.ec2.DescribeImages(&_ec2.DescribeImagesInput{
		Owners: []*string{types.StringP("282335181503")},
		Filters: []*_ec2.Filter{
			{
				Name: types.StringP("name"),
				Values: []*string{
					types.StringP("k8s-1.3-debian-jessie-amd64-hvm-ebs-2016-06-18"),
				},
			},
		},
	})
	if err != nil {
		return errors.FromErr(err).WithContext(conn.ctx).Err()
	}
	conn.ctx.InstanceImage = *r1.Images[0].ImageId
	conn.ctx.RootDeviceName = *r1.Images[0].RootDeviceName
	conn.ctx.Logger.Infof("Debain image with %v for %v detected", conn.ctx.InstanceImage, conn.ctx.RootDeviceName)
	return nil
}
