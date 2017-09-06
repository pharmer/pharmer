package hetzner

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	hc "github.com/appscode/go-hetzner"
	_ssh "github.com/appscode/go/crypto/ssh"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) Apply(cluster string, dryRun bool) error {
	var err error

	if cm.cluster, err = cloud.Store(cm.ctx).Clusters().Get(cluster); err != nil {
		return err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return err
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

	cm.cluster.Spec.Cloud.InstanceImage = "Debian 8.6 minimal"

	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	signer, err := _ssh.MakePrivateKeySignerFromBytes(cloud.SSHKey(cm.ctx).PrivateKey)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	cloud.Logger(cm.ctx).Info("Creating master instance")
	masterTx, err := im.createInstance(api.RoleKubernetesMaster, cm.cluster.Spec.MasterSKU)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterTx, err = cm.conn.waitForInstance(masterTx.ID, "ready")
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance, err := im.newKubeInstance(*masterTx.ServerIP)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Spec.Role = api.RoleKubernetesMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.Spec.MasterExternalIP)
	cloud.Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

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
	// cluster.Spec.UploadStartupConfig(cluster.Spec.ctx)

	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)

	cm.conn.client.Server.UpdateServer(&hc.ServerUpdateRequest{
		ServerIP:   *masterTx.ServerIP,
		ServerName: cm.cluster.Spec.KubernetesMasterName,
	})
	err = im.storeConfigFile(*masterTx.ServerIP, api.RoleKubernetesMaster, signer)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = im.storeStartupScript(*masterTx.ServerIP, cm.cluster.Spec.MasterSKU, api.RoleKubernetesMaster, signer)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = im.executeStartupScript(*masterTx.ServerIP, signer)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	//cluster.Spec.cloud.Logger(ctx).Info(">>>>>>>>>>>>>>>>>>>>>>> Rebooting master instance")
	//if err = cluster.Spec.reboot(masterDroplet.ID); err != nil {
	//	cluster.Spec.ctx.StatusCause = err.Error()
	//	return errors.FromErr(err).WithContext(cluster.Spec.ctx).Err()
	//}
	//cluster.Spec.cloud.Logger(ctx).Info(">>>>>>>>>>>>>>>>>>>>>>> Rebooted master instance")

	// start nodes
	//for _, ng := range req.NodeGroups {
	//	for i := int64(0); i < ng.Count; i++ {
	//		tx, err := im.createInstance(api.RoleKubernetesPool, ng.Sku)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//
	//		tx, err = cm.conn.waitForInstance(tx.ID, "ready")
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//		cm.conn.client.Server.UpdateServer(&hc.ServerUpdateRequest{
	//			ServerIP:   *tx.ServerIP,
	//			ServerName: cm.namer.GenNodeName(),
	//		})
	//
	//		err = im.storeConfigFile(*tx.ServerIP, api.RoleKubernetesPool, signer)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//		err = im.storeStartupScript(*tx.ServerIP, ng.Sku, api.RoleKubernetesPool, signer)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//		err = im.executeStartupScript(*tx.ServerIP, signer)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//
	//		// record nodes
	//		node, err := im.newKubeInstance(*tx.ServerIP)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//		node.Spec.Role = api.RoleKubernetesPool
	//		cloud.Store(cm.ctx).Instances(cm.cluster.Name).Create(node)
	//	}
	//}

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
	_, _, err := cm.conn.client.SSHKey.Create(&hc.SSHKeyCreateRequest{
		Name: cm.cluster.Name,
		Data: string(cloud.SSHKey(cm.ctx).PublicKey),
	})
	cloud.Logger(cm.ctx).Infof("New ssh key with fingerprint %v created", cloud.SSHKey(cm.ctx).OpensshFingerprint)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}
