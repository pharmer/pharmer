package vultr

import (
	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/appscode/pharmer/context"
	"github.com/appscode/pharmer/phid"
)

type clusterManager struct {
	ctx     context.Context
	cluster *api.Cluster
	ins     *api.ClusterInstances
	conn    *cloudConnector
	namer   namer
}

func (cm *clusterManager) initContext(req *proto.ClusterCreateRequest) error {
	err := cm.LoadDefaultContext()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.namer = namer{cluster: cm.cluster}

	//cluster.ctx.Name = req.Name
	//cluster.ctx.PHID = phid.NewKubeCluster()
	//cluster.ctx.Provider = req.Provider
	//cluster.ctx.Zone = req.Zone

	cm.cluster.Region = cm.cluster.Zone
	cm.cluster.DoNotDelete = req.DoNotDelete

	cm.cluster.SetNodeGroups(req.NodeGroups)

	cm.cluster.KubernetesMasterName = cm.namer.MasterName()
	cm.cluster.SSHKey, err = api.NewSSHKeyPair()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.SSHKeyExternalID = cm.namer.GenSSHKeyExternalID()
	cm.cluster.SSHKeyPHID = phid.NewSSHKey()

	cloud.GenClusterTokens(cm.cluster)

	return nil
}

func (cm *clusterManager) LoadDefaultContext() error {
	err := cloud.LoadDefaultGenericContext(cm.ctx, cm.cluster)
	if err != nil {
		return err
	}
	cm.cluster.OS = "debian"
	cm.cluster.MasterSKU = "94" // 2 cpu
	// Using custom image with memory controller enabled
	cm.cluster.InstanceImage = "16604964" // "container-os-20160402" // Debian 8.4 x64

	// https://discuss.vultr.com/discussion/197/what-is-the-meaning-of-enable-private-network
	cm.cluster.EnableClusterVPN = ""
	cm.cluster.VpnPsk = ""
	return nil
}

func (cm *clusterManager) UploadStartupConfig() error {
	if api.UseFirebase() {
		return cloud.UploadStartupConfigInFirebase(cm.ctx, cm.cluster)
	}
	return nil
}
