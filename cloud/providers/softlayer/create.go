package softlayer

import (
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/types"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	"github.com/softlayer/softlayer-go/datatypes"
)

func (cm *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	var err error

	cm.cluster, err = NewCluster(req)
	if err != nil {
		return err
	}
	cm.namer = namer{cluster: cm.cluster}

	if _, err := cloud.Store(cm.ctx).Clusters().Create(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.conn, err = NewConnector(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	defer func(releaseReservedIp bool) {
		if cm.cluster.Status.Phase == api.ClusterPhasePending {
			cm.cluster.Status.Phase = api.ClusterPhaseFailing
		}
		cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		cloud.Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterPhaseReady {
			cloud.Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.Delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")
	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	masterId, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleKubernetesMaster, cm.cluster.Spec.MasterSKU)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn.waitForInstance(masterId)

	masterInstance, err := im.newKubeInstance(masterId)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Spec.Role = api.RoleKubernetesMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
	cloud.Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

	if cm.ctx, err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// needed to get master_internal_ip
	if _, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// --------------------------------------------------------------------

	time.Sleep(60 * time.Second)

	cloud.Logger(cm.ctx).Info("Rebooting master instance")
	if _, err = im.reboot(masterId); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Info("Rebooted master instance")

	// start nodes
	for _, ng := range req.NodeGroups {
		for i := int64(0); i < ng.Count; i++ {
			serverID, err := im.createInstance(cm.namer.GenNodeName(), api.RoleKubernetesPool, ng.Sku)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			// record nodes
			cm.conn.waitForInstance(serverID)
			node, err := im.newKubeInstance(serverID)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			node.Spec.Role = api.RoleKubernetesPool
			node.Spec.SKU = ng.Sku
			cloud.Store(cm.ctx).Instances(cm.cluster.Name).Create(node)
		}
	}

	cloud.Logger(cm.ctx).Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := cloud.EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err := cloud.WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Status.Phase = api.ClusterPhaseReady
	return nil
}

func (cm *ClusterManager) importPublicKey() error {
	cloud.Logger(cm.ctx).Debugln("Adding SSH public key")

	securitySSHTemplate := datatypes.Security_Ssh_Key{
		Label: types.StringP(cm.cluster.Name),
		Key:   types.StringP(string(cm.cluster.Spec.SSHKey.PublicKey)),
	}

	backoff.Retry(func() error {
		sk, err := cm.conn.securityServiceClient.CreateObject(&securitySSHTemplate)

		cm.cluster.Spec.SSHKeyExternalID = strconv.Itoa(*sk.Id)
		return err
	}, backoff.NewExponentialBackOff())
	cloud.Logger(cm.ctx).Debugf("Created new ssh key with fingerprint=%v", cm.cluster.Spec.SSHKey.OpensshFingerprint)
	return nil
}
