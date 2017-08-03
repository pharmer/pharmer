package linode

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/common"
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
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.OS = "debian"
	cm.ctx.InstanceRootPassword = "@dmin123"
	if _env.FromHost().IsPublic() {
		cm.ctx.InstanceRootPassword = rand.GeneratePassword()
	}
	cm.ctx.MasterSKU = "2" // plan_id 2 label {"Linode 4096"} cpu 2 ram 4096 disk 48

	common.BuildRuntimeConfig(cm.ctx)
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	if api.UseFirebase() {
		return common.UploadStartupConfigInFirebase(cm.ctx)
	}
	return nil
}
