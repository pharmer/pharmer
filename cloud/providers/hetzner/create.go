package hetzner

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	hc "github.com/appscode/go-hetzner"
	_ssh "github.com/appscode/go/crypto/ssh"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
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
	cm.conn, err = NewConnector(cm.ctx, cm.cluster)
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

	cm.cluster.InstanceImage = "Debian 8.6 minimal"

	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	signer, err := _ssh.MakePrivateKeySignerFromBytes(cm.cluster.SSHKey.PrivateKey)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn}

	cm.ctx.Logger().Info("Creating master instance")
	masterTx, err := im.createInstance(api.RoleKubernetesMaster, cm.cluster.MasterSKU)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterTx, err = cm.conn.waitForInstance(masterTx.ID, "ready")
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance, err := im.newKubeInstance(*masterTx.ServerIP)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Role = api.RoleKubernetesMaster
	cm.cluster.MasterExternalIP = masterInstance.ExternalIP
	cm.cluster.MasterInternalIP = masterInstance.InternalIP
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.MasterExternalIP)
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
	// cluster.UploadStartupConfig(cluster.ctx)

	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)

	cm.conn.client.Server.UpdateServer(&hc.ServerUpdateRequest{
		ServerIP:   *masterTx.ServerIP,
		ServerName: cm.cluster.KubernetesMasterName,
	})
	err = im.storeConfigFile(*masterTx.ServerIP, api.RoleKubernetesMaster, signer)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = im.storeStartupScript(*masterTx.ServerIP, cm.cluster.MasterSKU, api.RoleKubernetesMaster, signer)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = im.executeStartupScript(*masterTx.ServerIP, signer)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	//cluster.ctx.Logger().Info(">>>>>>>>>>>>>>>>>>>>>>> Rebooting master instance")
	//if err = cluster.reboot(masterDroplet.ID); err != nil {
	//	cluster.ctx.StatusCause = err.Error()
	//	return errors.FromErr(err).WithContext(cluster.ctx).Err()
	//}
	//cluster.ctx.Logger().Info(">>>>>>>>>>>>>>>>>>>>>>> Rebooted master instance")

	// start nodes
	for sku, count := range req.NodeSet {
		for i := int64(0); i < count; i++ {
			tx, err := im.createInstance(api.RoleKubernetesPool, sku)
			if err != nil {
				cm.cluster.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			tx, err = cm.conn.waitForInstance(tx.ID, "ready")
			if err != nil {
				cm.cluster.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			cm.conn.client.Server.UpdateServer(&hc.ServerUpdateRequest{
				ServerIP:   *tx.ServerIP,
				ServerName: cm.namer.GenNodeName(),
			})

			err = im.storeConfigFile(*tx.ServerIP, api.RoleKubernetesPool, signer)
			if err != nil {
				cm.cluster.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			err = im.storeStartupScript(*tx.ServerIP, sku, api.RoleKubernetesPool, signer)
			if err != nil {
				cm.cluster.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			err = im.executeStartupScript(*tx.ServerIP, signer)
			if err != nil {
				cm.cluster.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			// record nodes
			node, err := im.newKubeInstance(*tx.ServerIP)
			if err != nil {
				cm.cluster.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			node.Role = api.RoleKubernetesPool
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
	_, _, err := cm.conn.client.SSHKey.Create(&hc.SSHKeyCreateRequest{
		Name: cm.cluster.Name,
		Data: string(cm.cluster.SSHKey.PublicKey),
	})
	cm.ctx.Logger().Infof("New ssh key with fingerprint %v created", cm.cluster.SSHKey.OpensshFingerprint)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}
