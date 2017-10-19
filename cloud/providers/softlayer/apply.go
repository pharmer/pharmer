package softlayer

import (
	"strconv"
	"time"

	"github.com/appscode/go/errors"
	. "github.com/appscode/go/types"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	"github.com/softlayer/softlayer-go/datatypes"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error

	cm.cluster = in
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}

	//defer func(releaseReservedIp bool) {
	//	if cm.cluster.Status.Phase == api.ClusterPending {
	//		cm.cluster.Status.Phase = api.ClusterFailing
	//	}
	//	Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	//	Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
	//	if cm.cluster.Status.Phase != api.ClusterReady {
	//		Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
	//		cm.Delete(&proto.ClusterDeleteRequest{
	//			Name:              cm.cluster.Name,
	//			ReleaseReservedIp: releaseReservedIp,
	//		})
	//	}
	//}(cm.cluster.Spec.MasterReservedIP == "auto")
	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	masterId, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleMaster, cm.cluster.Spec.MasterSKU)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn.waitForInstance(masterId)

	masterInstance, err := im.newKubeInstance(masterId)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Spec.Role = api.RoleMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
	Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

	err = EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// needed to get master_internal_ip
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// --------------------------------------------------------------------

	time.Sleep(60 * time.Second)

	Logger(cm.ctx).Info("Rebooting master instance")
	if _, err = im.reboot(masterId); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Info("Rebooted master instance")

	// start nodes
	//for _, ng := range req.NodeGroups {
	//	for i := int64(0); i < ng.Count; i++ {
	//		serverID, err := im.createInstance(cm.namer.GenNodeName(), api.RoleKubernetesPool, ng.Sku)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//		// record nodes
	//		cm.conn.waitForInstance(serverID)
	//		node, err := im.newKubeInstance(serverID)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//		node.Spec.Role = api.RoleKubernetesPool
	//		node.Spec.SKU = ng.Sku
	//		Store(cm.ctx).Instances(cm.cluster.Name).Create(node)
	//	}
	//}

	Logger(cm.ctx).Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	kc, err := cm.GetAdminClient()
	// wait for nodes to start
	if err := WaitForReadyMaster(cm.ctx, kc); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.Status.Phase = api.ClusterReady
	return nil, nil
}

func (cm *ClusterManager) importPublicKey() error {
	Logger(cm.ctx).Debugln("Adding SSH public key")

	securitySSHTemplate := datatypes.Security_Ssh_Key{
		Label: StringP(cm.cluster.Name),
		Key:   StringP(string(SSHKey(cm.ctx).PublicKey)),
	}

	backoff.Retry(func() error {
		sk, err := cm.conn.securityServiceClient.CreateObject(&securitySSHTemplate)

		cm.cluster.Status.SSHKeyExternalID = strconv.Itoa(*sk.Id)
		return err
	}, backoff.NewExponentialBackOff())
	Logger(cm.ctx).Debugf("Created new ssh key with fingerprint=%v", SSHKey(cm.ctx).OpensshFingerprint)
	return nil
}
