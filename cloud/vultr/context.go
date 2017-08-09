package vultr

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/common"
	"github.com/appscode/pharmer/contexts"
	"github.com/appscode/pharmer/phid"
)

type clusterManager struct {
	ctx   *contexts.ClusterContext
	ins   *contexts.ClusterInstances
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

	cm.ctx.Region = cm.ctx.Zone
	cm.ctx.DoNotDelete = req.DoNotDelete
	common.SetApps(cm.ctx)

	cm.ctx.SetNodeGroups(req.NodeGroups)

	cm.ctx.KubernetesMasterName = cm.namer.MasterName()
	cm.ctx.SSHKey, err = contexts.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.ctx.SSHKeyPHID = phid.NewSSHKey()

	common.GenClusterTokens(cm.ctx)

	cm.ctx.AppsCodeNamespace = cm.ctx.Auth.Namespace

	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := common.LoadDefaultGenericContext(cm.ctx)
	if err != nil {
		return err
	}
	cm.ctx.OS = "debian"
	cm.ctx.MasterSKU = "94" // 2 cpu
	// Using custom image with memory controller enabled
	cm.ctx.InstanceImage = "16604964" // "container-os-20160402" // Debian 8.4 x64

	// https://discuss.vultr.com/discussion/197/what-is-the-meaning-of-enable-private-network
	cm.ctx.EnableClusterVPN = ""
	cm.ctx.VpnPsk = ""
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	if api.UseFirebase() {
		return common.UploadStartupConfigInFirebase(cm.ctx)
	}
	return nil
}
