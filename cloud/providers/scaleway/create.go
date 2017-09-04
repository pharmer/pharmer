package scaleway

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/crypto/ssh"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
)

func (cm *ClusterManager) Create(req *proto.ClusterCreateRequest) error {
	err := cm.NewCluster(req)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn, err = NewConnector(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	defer func(releaseReservedIP bool) {
		if cm.cluster.Status.Phase == api.ClusterPhasePending {
			cm.cluster.Status.Phase = api.ClusterPhaseFailing
		}
		cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
		cloud.Logger(cm.ctx).Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterPhaseReady {
			cloud.Logger(cm.ctx).Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.Delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIP,
			})
		}
	}(cm.cluster.Spec.MasterReservedIP == "auto")

	cm.cluster.Spec.InstanceImage, err = cm.conn.getInstanceImage()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Using image id %v", cm.cluster.Spec.InstanceImage)

	err = cm.conn.DetectBootscript()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Using bootscript id %v", cm.conn.bootscriptID)

	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	signer, err := ssh.MakePrivateKeySignerFromBytes(cm.cluster.Spec.SSHKey.PrivateKey)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterIPID, err := cm.reserveIP()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if cm.ctx, err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if _, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	cloud.Logger(cm.ctx).Info("Creating master instance")
	masterID, err := im.createInstance(cm.cluster.Spec.KubernetesMasterName, api.RoleKubernetesMaster, cm.cluster.Spec.MasterSKU, masterIPID)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.conn.waitForInstance(masterID, "running"); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterInstance, err := im.newKubeInstance(masterID)
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
	if _, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Saved cluster context with MASTER_INTERNAL_IP")

	// reboot master to use cert with internal_ip as SANS
	time.Sleep(30 * time.Second)

	err = im.executeStartupScript(masterInstance, signer)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// start nodes
	for _, ng := range req.NodeGroups {
		for i := int64(0); i < ng.Count; i++ {
			serverID, err := im.createInstance(cm.namer.GenNodeName(), api.RoleKubernetesPool, ng.Sku)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			// record nodes
			cm.conn.waitForInstance(serverID, "running")
			node, err := im.newKubeInstance(serverID)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			node.Spec.Role = api.RoleKubernetesPool
			cloud.Store(cm.ctx).Instances(cm.cluster.Name).Create(node)

			err = im.executeStartupScript(node, signer)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
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
	cloud.Logger(cm.ctx).Infof("Adding SSH public key")
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
			Key: string(cm.cluster.Spec.SSHKey.PublicKey),
		}

		return cm.conn.client.PatchUserSSHKey(user.ID, sapi.ScalewayUserPatchSSHKeyDefinition{
			SSHPublicKeys: sshPubKeys,
		})
	}, backoff.NewExponentialBackOff())
	cloud.Logger(cm.ctx).Infof("New ssh key with fingerprint %v created", cm.cluster.Spec.SSHKey.OpensshFingerprint)
	return nil
}

func (cm *ClusterManager) reserveIP() (string, error) {
	// if cluster.Spec.ctx.MasterReservedIP == "auto" {
	cloud.Logger(cm.ctx).Infof("Reserving Floating IP")
	fip, err := cm.conn.client.NewIP()
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("New floating ip %v reserved", fip.IP)
	cm.cluster.Spec.MasterReservedIP = fip.IP.Address
	return fip.IP.ID, nil
	// }
	// return "", nil
}
