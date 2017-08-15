package hetzner

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	hc "github.com/appscode/go-hetzner"
	_ssh "github.com/appscode/go/crypto/ssh"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = lib.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn, err = NewConnector(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	defer func(releaseReservedIp bool) {
		if cm.ctx.Status == storage.KubernetesStatus_Pending {
			cm.ctx.Status = storage.KubernetesStatus_Failing
		}
		cm.ctx.Save()
		cm.ins.Save()
		cm.ctx.Logger().Infof("Cluster %v is %v", cm.ctx.Name, cm.ctx.Status)
		if cm.ctx.Status != storage.KubernetesStatus_Ready {
			cm.ctx.Logger().Infof("Cluster %v is deleting", cm.ctx.Name)
			cm.delete(&proto.ClusterDeleteRequest{
				Name:              cm.ctx.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.ctx.MasterReservedIP == "auto")

	cm.ctx.InstanceImage = "Debian 8.6 minimal"

	err = cm.importPublicKey()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	signer, err := _ssh.MakePrivateKeySignerFromBytes(cm.ctx.SSHKey.PrivateKey)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn}

	cm.ctx.Logger().Info("Creating master instance")
	masterTx, err := im.createInstance(system.RoleKubernetesMaster, cm.ctx.MasterSKU)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterTx, err = cm.conn.waitForInstance(masterTx.ID, "ready")
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance, err := im.newKubeInstance(*masterTx.ServerIP)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Role = system.RoleKubernetesMaster
	cm.ctx.MasterExternalIP = masterInstance.ExternalIP
	cm.ctx.MasterInternalIP = masterInstance.InternalIP
	fmt.Println("Master EXTERNAL IP ================", cm.ctx.MasterExternalIP)
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	if err = lib.GenClusterCerts(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = lib.EnsureARecord(cm.ctx, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.DetectApiServerURL()
	// needed to get master_internal_ip
	if err = cm.ctx.Save(); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// cluster.UploadStartupConfig(cluster.ctx)

	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)

	cm.conn.client.Server.UpdateServer(&hc.ServerUpdateRequest{
		ServerIP:   *masterTx.ServerIP,
		ServerName: cm.ctx.KubernetesMasterName,
	})
	err = im.storeConfigFile(*masterTx.ServerIP, system.RoleKubernetesMaster, signer)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = im.storeStartupScript(*masterTx.ServerIP, cm.ctx.MasterSKU, system.RoleKubernetesMaster, signer)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = im.executeStartupScript(*masterTx.ServerIP, signer)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
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
			tx, err := im.createInstance(system.RoleKubernetesPool, sku)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			tx, err = cm.conn.waitForInstance(tx.ID, "ready")
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			cm.conn.client.Server.UpdateServer(&hc.ServerUpdateRequest{
				ServerIP:   *tx.ServerIP,
				ServerName: cm.namer.GenNodeName(),
			})

			err = im.storeConfigFile(*tx.ServerIP, system.RoleKubernetesPool, signer)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			err = im.storeStartupScript(*tx.ServerIP, sku, system.RoleKubernetesPool, signer)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			err = im.executeStartupScript(*tx.ServerIP, signer)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}

			// record nodes
			node, err := im.newKubeInstance(*tx.ServerIP)
			if err != nil {
				cm.ctx.StatusCause = err.Error()
				return errors.FromErr(err).WithContext(cm.ctx).Err()
			}
			node.Role = system.RoleKubernetesPool
			cm.ins.Instances = append(cm.ins.Instances, node)
		}
	}

	cm.ctx.Logger().Info("Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := lib.EnsureDnsIPLookup(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err := lib.ProbeKubeAPI(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = lib.CheckComponentStatuses(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = lib.WaitForReadyNodes(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Status = storage.KubernetesStatus_Ready
	return nil
}

func (cm *clusterManager) importPublicKey() error {
	_, _, err := cm.conn.client.SSHKey.Create(&hc.SSHKeyCreateRequest{
		Name: cm.ctx.Name,
		Data: string(cm.ctx.SSHKey.PublicKey),
	})
	cm.ctx.Logger().Infof("New ssh key with fingerprint %v created", cm.ctx.SSHKey.OpensshFingerprint)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}
