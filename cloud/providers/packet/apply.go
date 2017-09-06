package packet

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	"github.com/packethost/packngo"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) error {
	var err error

	cm.cluster = in
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

	cm.cluster.Spec.Cloud.InstanceImage = "debian_8"

	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	cloud.Logger(cm.ctx).Info("Creating master instance")
	masterDroplet, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleKubernetesMaster, cm.cluster.Spec.MasterSKU)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.conn.waitForInstance(masterDroplet.ID, "active"); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Need to reload te master server Object to get the IP address
	masterInstance, err := im.newKubeInstance(masterDroplet.ID)
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

	// -------------------------- NODES

	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)

	cloud.Logger(cm.ctx).Info("Rebooting master instance")
	if err = im.reboot(masterDroplet.ID); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Info("Rebooted master instance")

	// start nodes
	//for _, ng := range req.NodeGroups {
	//	for i := int64(0); i < ng.Count; i++ {
	//		droplet, err := im.createInstance(cm.namer.GenNodeName(), api.RoleKubernetesPool, ng.Sku)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//
	//		// record nodes
	//		cm.conn.waitForInstance(droplet.ID, "active")
	//		node, err := im.newKubeInstanceFromServer(droplet)
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
	cloud.Logger(cm.ctx).Debugln("Adding SSH public key")
	backoff.Retry(func() error {
		sk, _, err := cm.conn.client.SSHKeys.Create(&packngo.SSHKeyCreateRequest{
			Key:       string(cloud.SSHKey(cm.ctx).PublicKey),
			Label:     cm.cluster.Name,
			ProjectID: cm.cluster.Spec.Cloud.Project,
		})
		cm.cluster.Status.SSHKeyExternalID = sk.ID
		return err
	}, backoff.NewExponentialBackOff())
	cloud.Logger(cm.ctx).Debugf("Created new ssh key with fingerprint=%v", cloud.SSHKey(cm.ctx).OpensshFingerprint)
	return nil
}
