package digitalocean

import (
	"context"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	// Deprecated
	namer namer
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "digitalocean"
)

func init() {
	cloud.RegisterCloudManager(UID, func(ctx context.Context) (cloud.Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.Interface {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) GetInstance(md *api.NodeStatus) (*api.Node, error) {
	conn, err := NewConnector(cm.ctx, nil)
	if err != nil {
		return nil, err
	}
	im := &instanceManager{conn: conn}
	return im.GetInstance(md)
}

func (cm *ClusterManager) MatchInstance(i *api.Node, md *api.NodeStatus) bool {
	return i.Status.PrivateIP == md.PrivateIP
}
