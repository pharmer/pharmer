package digitalocean

import (
	"context"

	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	conn    *cloudConnector
	// Deprecated
	namer namer
}

var _ Interface = &ClusterManager{}

const (
	UID = "digitalocean"
)

func init() {
	RegisterCloudManager(UID, func(ctx context.Context) (Interface, error) { return New(ctx), nil })
}

func New(ctx context.Context) Interface {
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
