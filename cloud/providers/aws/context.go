package aws

import (
	"context"
	"time"

	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	_ec2 "github.com/aws/aws-sdk-go/service/ec2"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	// Deprecated
	namer namer
}

var _ cloud.ClusterManager = &ClusterManager{}

const (
	UID = "aws"
)

func init() {
	cloud.RegisterCloudManager(UID, func(ctx context.Context) (cloud.ClusterManager, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.ClusterManager {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) GetInstance(md *api.NodeStatus) (*api.Node, error) {
	conn, err := NewConnector(cm.ctx, cm.cluster)
	if err != nil {
		return nil, err
	}
	cm.conn = conn
	i, err := cm.newKubeInstance(md.ExternalID)
	if err != nil {
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// TODO: Role not set
	return i, nil
}

func (cm *ClusterManager) MatchInstance(i *api.Node, md *api.NodeStatus) bool {
	return i.Status.ExternalID == md.ExternalID
}

func (cm *ClusterManager) waitForInstanceState(instanceId string, state string) error {
	for {
		r1, err := cm.conn.ec2.DescribeInstances(&_ec2.DescribeInstancesInput{
			InstanceIds: []*string{StringP(instanceId)},
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		curState := *r1.Reservations[0].Instances[0].State.Name
		if curState == state {
			break
		}
		cloud.Logger(cm.ctx).Infof("Waiting for instance %v to be %v (currently %v)", instanceId, state, curState)
		cloud.Logger(cm.ctx).Infof("Sleeping for 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	return nil
}
