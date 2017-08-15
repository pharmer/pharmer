package softlayer

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/phid"
)

type clusterManager struct {
	ctx   *api.Cluster
	ins   *api.ClusterInstances
	conn  *cloudConnector
	namer namer
}

func (cm *clusterManager) initContext(req *proto.ClusterCreateRequest) error {
	err := cm.LoadDefaultContext()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.namer = namer{ctx: cm.ctx}

	//cluster.ctx.Name = req.Name
	//cluster.ctx.PHID = phid.NewKubeCluster()
	//cluster.ctx.Provider = req.Provider
	//cluster.ctx.Zone = req.Zone

	cm.ctx.Region = cm.ctx.Zone[:len(cm.ctx.Zone)-2]
	cm.ctx.DoNotDelete = req.DoNotDelete

	cm.ctx.SetNodeGroups(req.NodeGroups)

	cm.ctx.KubernetesMasterName = cm.namer.MasterName()
	cm.ctx.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.ctx.SSHKeyPHID = phid.NewSSHKey()
	cloud.GenClusterTokens(cm.ctx)

	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := cloud.LoadDefaultGenericContext(cm.ctx)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.OS = "DEBIAN_8_64"
	cm.ctx.MasterSKU = "1c2m"

	cm.ctx.EnableClusterVPN = ""
	cm.ctx.VpnPsk = ""
	return nil
}

func (cm *clusterManager) UploadStartupConfig(ctx *api.Cluster) error {
	if api.UseFirebase() {
		return cloud.UploadStartupConfigInFirebase(ctx)
	}
	return nil
}
