package softlayer

import (
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	"github.com/softlayer/softlayer-go/datatypes"
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn, err = NewConnector(cm.cluster)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	defer func(releaseReservedIp bool) {
		if cm.cluster.Status == api.KubernetesStatus_Pending {
			cm.cluster.Status = api.KubernetesStatus_Failing
		}
		cm.ctx.Store().Clusters().SaveCluster(cm.cluster)
		cm.ctx.Store().Instances().SaveInstances(cm.ins.Instances)
		cm.ctx.Logger().Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status)
		if cm.cluster.Status != api.KubernetesStatus_Ready {
			cm.ctx.Logger().Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.MasterReservedIP == "auto")
	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	masterId, err := im.createInstance(cm.cluster.KubernetesMasterName, api.RoleKubernetesMaster, cm.cluster.MasterSKU)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn.waitForInstance(masterId)

	masterInstance, err := im.newKubeInstance(masterId)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Role = api.RoleKubernetesMaster
	cm.cluster.MasterExternalIP = masterInstance.ExternalIP
	cm.cluster.MasterInternalIP = masterInstance.InternalIP
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	if err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.cluster.DetectApiServerURL()
	// needed to get master_internal_ip
	if err = cm.ctx.Store().Clusters().SaveCluster(cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.UploadStartupConfig(cm.cluster)

	// --------------------------------------------------------------------

	time.Sleep(60 * time.Second)

	cm.ctx.Logger().Info("Rebooting master instance")
	if _, err = im.reboot(masterId); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Info("Rebooted master instance")

	// start nodes
	for _, ng := range req.NodeGroups {
		for i := int64(0); i < ng.Count; i++ {
			serverID, err := im.createInstance(cm.namer.GenNodeName(), api.RoleKubernetesPool, ng.Sku)
			if err != nil {
				cm.cluster.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			// record nodes
			cm.conn.waitForInstance(serverID)
			node, err := im.newKubeInstance(serverID)
			if err != nil {
				cm.cluster.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			node.Role = api.RoleKubernetesPool
			node.SKU = ng.Sku
			cm.ins.Instances = append(cm.ins.Instances, node)
		}
	}

	cm.ctx.Logger().Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := cloud.EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err := cloud.ProbeKubeAPI(cm.ctx, cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = cloud.CheckComponentStatuses(cm.ctx, cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = cloud.WaitForReadyNodes(cm.ctx, cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.cluster.Status = api.KubernetesStatus_Ready
	return nil
}

func (cm *clusterManager) importPublicKey() error {
	cm.ctx.Logger().Debugln("Adding SSH public key")

	securitySSHTemplate := datatypes.Security_Ssh_Key{
		Label: types.StringP(cm.cluster.Name),
		Key:   types.StringP(string(cm.cluster.SSHKey.PublicKey)),
	}

	backoff.Retry(func() error {
		sk, err := cm.conn.securityServiceClient.CreateObject(&securitySSHTemplate)

		cm.cluster.SSHKeyExternalID = strconv.Itoa(*sk.Id)
		return err
	}, backoff.NewExponentialBackOff())
	cm.ctx.Logger().Debugf("Created new ssh key with fingerprint=%v", cm.cluster.SSHKey.OpensshFingerprint)
	return nil
}