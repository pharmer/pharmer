package scaleway

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/ssh"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error

	cm.cluster = in
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}

	//defer func(releaseReservedIP bool) {
	//	if cm.cluster.Status.Phase == api.ClusterPending {
	//		cm.cluster.Status.Phase = api.ClusterFailing
	//	}
	//	Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	//	Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
	//	if cm.cluster.Status.Phase != api.ClusterReady {
	//		Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
	//		cm.Delete(&proto.ClusterDeleteRequest{
	//			Name:              cm.cluster.Name,
	//			ReleaseReservedIp: releaseReservedIP,
	//		})
	//	}
	//}(cm.cluster.Spec.MasterReservedIP == "auto")

	cm.cluster.Spec.Cloud.InstanceImage, err = cm.conn.getInstanceImage()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Using image id %v", cm.cluster.Spec.Cloud.InstanceImage)

	err = cm.conn.DetectBootscript()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Using bootscript id %v", cm.conn.bootscriptID)

	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	signer, err := ssh.MakePrivateKeySignerFromBytes(SSHKey(cm.ctx).PrivateKey)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterIPID, err := cm.reserveIP()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	Logger(cm.ctx).Info("Creating master instance")
	masterID, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleMaster, cm.cluster.Spec.MasterSKU, masterIPID)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.conn.waitForInstance(masterID, "running"); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterInstance, err := im.newKubeInstance(masterID)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Spec.Role = api.RoleMaster
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.PublicIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.PrivateIP
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.Spec.MasterExternalIP)
	Store(cm.ctx).Instances(cm.cluster.Name).Create(masterInstance)

	err = EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("Saved cluster context with MASTER_INTERNAL_IP")

	// reboot master to use cert with internal_ip as SANS
	time.Sleep(30 * time.Second)

	err = im.executeStartupScript(masterInstance, signer)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// start nodes
	//for _, ng := range req.NodeGroups {
	//	for i := int64(0); i < ng.Count; i++ {
	//		serverID, err := im.createInstance(cm.namer.GenNodeName(), api.RoleKubernetesPool, ng.Sku)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//
	//		// record nodes
	//		cm.conn.waitForInstance(serverID, "running")
	//		node, err := im.newKubeInstance(serverID)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
	//		node.Spec.Role = api.RoleKubernetesPool
	//		Store(cm.ctx).Instances(cm.cluster.Name).Create(node)
	//
	//		err = im.executeStartupScript(node, signer)
	//		if err != nil {
	//			cm.cluster.Status.Reason = err.Error()
	//			return nil, errors.FromErr(err).WithContext(cm.ctx).Err()
	//		}
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
	Logger(cm.ctx).Infof("Adding SSH public key")
	backoff.Retry(func() error {
		user, err := cm.conn.client.GetUser()
		if err != nil {
			return err
		}

		sshPubKeys := make([]sapi.ScalewayKeyDefinition, len(user.SSHPublicKeys)+1)
		for i, kk := range user.SSHPublicKeys {
			sshPubKeys[i] = sapi.ScalewayKeyDefinition{Key: kk.Key}
		}
		sshPubKeys[len(user.SSHPublicKeys)] = sapi.ScalewayKeyDefinition{
			Key: string(SSHKey(cm.ctx).PublicKey),
		}

		return cm.conn.client.PatchUserSSHKey(user.ID, sapi.ScalewayUserPatchSSHKeyDefinition{
			SSHPublicKeys: sshPubKeys,
		})
	}, backoff.NewExponentialBackOff())
	Logger(cm.ctx).Infof("New ssh key with fingerprint %v created", SSHKey(cm.ctx).OpensshFingerprint)
	return nil
}

func (cm *ClusterManager) reserveIP() (string, error) {
	// if cluster.Spec.ctx.MasterReservedIP == "auto" {
	Logger(cm.ctx).Infof("Reserving Floating IP")
	fip, err := cm.conn.client.NewIP()
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	Logger(cm.ctx).Infof("New floating ip %v reserved", fip.IP)
	cm.cluster.Spec.MasterReservedIP = fip.IP.Address
	return fip.IP.ID, nil
	// }
	// return "", nil
}
