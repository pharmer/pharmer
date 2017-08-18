package scaleway

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/ssh"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
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
		cm.ctx.Store().Clusters().SaveCluster(cm.cluster)
		cm.ctx.Store().Instances().SaveInstances(cm.ins.Instances)
		cm.ctx.Logger().Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status.Phase)
		if cm.cluster.Status.Phase != api.ClusterPhaseReady {
			cm.ctx.Logger().Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.delete(&proto.ClusterDeleteRequest{
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
	cm.ctx.Logger().Infof("Using image id %v", cm.cluster.Spec.InstanceImage)

	err = cm.conn.DetectBootscript()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Using bootscript id %v", cm.conn.bootscriptID)

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

	cm.cluster.DetectApiServerURL()
	if err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.ctx.Store().Clusters().SaveCluster(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	cm.ctx.Logger().Info("Creating master instance")
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
	cm.cluster.Spec.MasterExternalIP = masterInstance.Status.ExternalIP
	cm.cluster.Spec.MasterInternalIP = masterInstance.Status.InternalIP
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.Spec.MasterExternalIP)
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.ctx.Store().Clusters().SaveCluster(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Saved cluster context with MASTER_INTERNAL_IP")

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
			cm.ins.Instances = append(cm.ins.Instances, node)

			err = im.executeStartupScript(node, signer)
			if err != nil {
				cm.cluster.Status.Reason = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
		}
	}

	cm.ctx.Logger().Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := cloud.EnsureDnsIPLookup(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err := cloud.ProbeKubeAPI(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = cloud.CheckComponentStatuses(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = cloud.WaitForReadyNodes(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.cluster.Status.Phase = api.ClusterPhaseReady
	return nil
}

func (cm *clusterManager) importPublicKey() error {
	cm.ctx.Logger().Infof("Adding SSH public key")
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
	cm.ctx.Logger().Infof("New ssh key with fingerprint %v created", cm.cluster.Spec.SSHKey.OpensshFingerprint)
	return nil
}

func (cm *clusterManager) reserveIP() (string, error) {
	// if cluster.Spec.ctx.MasterReservedIP == "auto" {
	cm.ctx.Logger().Infof("Reserving Floating IP")
	fip, err := cm.conn.client.NewIP()
	if err != nil {
		return "", errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("New floating ip %v reserved", fip.IP)
	cm.cluster.Spec.MasterReservedIP = fip.IP.Address
	return fip.IP.ID, nil
	// }
	// return "", nil
}
