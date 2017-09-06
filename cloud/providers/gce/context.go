package gce

import (
	"context"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

const (
	maxInstancesPerMIG = 5 // Should be 500
	defaultNetwork     = "default"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	namer   namer
}

var _ cloud.ClusterManager = &ClusterManager{}

const (
	UID = "gce"
)

func init() {
	cloud.RegisterCloudManager(UID, func(ctx context.Context) (cloud.ClusterManager, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.ClusterManager {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) MatchInstance(i *api.Instance, md *api.InstanceStatus) bool {
	return i.Name == md.Name
}
