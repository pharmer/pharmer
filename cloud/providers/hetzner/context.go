package hetzner

import (
	"context"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
)

type ClusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	ins     *api.ClusterInstances
	conn    *cloudConnector
	namer   namer
}

var _ cloud.ClusterManager = &ClusterManager{}

const (
	UID = "hetzner"
)

func init() {
	cloud.RegisterCloudManager(UID, func(ctx context.Context) (cloud.ClusterManager, error) { return New(ctx), nil })
}

func New(ctx context.Context) cloud.ClusterManager {
	return &ClusterManager{ctx: ctx}
}

func (cm *ClusterManager) Scale(req *proto.ClusterReconfigureRequest) error {
	return cloud.UnsupportedOperation
}

func (cm *ClusterManager) SetVersion(req *proto.ClusterReconfigureRequest) error {
	return cloud.UnsupportedOperation
}

func (cm *ClusterManager) UploadStartupConfig() error {
	return cloud.UnsupportedOperation
}

func (cm *ClusterManager) GetInstance(md *api.InstanceMetadata) (*api.Instance, error) {
	conn, err := NewConnector(cm.ctx, nil)
	if err != nil {
		return nil, err
	}
	im := &instanceManager{conn: conn}
	return im.GetInstance(md)
}

func (cm *ClusterManager) MatchInstance(i *api.Instance, md *api.InstanceMetadata) bool {
	return i.Status.InternalIP == md.ExternalIP
}

func (cm *ClusterManager) initContext(req *proto.ClusterCreateRequest) error {
	err := cm.LoadDefaultContext()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.namer = namer{cluster: cm.cluster}

	//cluster.Spec.ctx.Name = req.Name
	//cluster.Spec.ctx.PHID = phid.NewKubeCluster()
	//cluster.Spec.ctx.Provider = req.Provider
	//cluster.Spec.ctx.Zone = req.Zone

	cm.cluster.Spec.Region = cm.cluster.Spec.Zone
	cm.cluster.Spec.DoNotDelete = req.DoNotDelete

	cm.cluster.SetNodeGroups(req.NodeGroups)

	cm.cluster.Spec.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.Spec.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Spec.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.Spec.SSHKeyPHID = phid.NewSSHKey()

	return nil
}

func (cm *ClusterManager) LoadDefaultContext() error {
	err := cloud.LoadDefaultGenericContext(cm.ctx, cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.cluster.Spec.OS = "debian"
	cm.cluster.Spec.MasterSKU = "CX10" // CX20

	cloud.BuildRuntimeConfig(cm.cluster)
	return nil
}
