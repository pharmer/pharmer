package gce

import (
	"context"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
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

var _ Interface = &ClusterManager{}

const (
	UID = "gce"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) MatchInstance(i *api.Node, md *api.NodeStatus) bool {
	return i.Name == md.Name
}
