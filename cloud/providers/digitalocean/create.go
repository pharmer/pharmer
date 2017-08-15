package digitalocean

import (
	go_ctx "context"
	"fmt"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/digitalocean/godo"
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
	cm.cluster.Save()

	defer func(releaseReservedIp bool) {
		if cm.cluster.Status == api.KubernetesStatus_Pending {
			cm.cluster.Status = api.KubernetesStatus_Failing
		}
		cm.cluster.Save()
		cm.ins.Save()
		cm.ctx.Logger().Infof("Cluster %v is %v", cm.cluster.Name, cm.cluster.Status)
		if cm.cluster.Status != api.KubernetesStatus_Ready {
			cm.ctx.Logger().Infof("Cluster %v is deleting", cm.cluster.Name)
			cm.delete(&proto.ClusterDeleteRequest{
				Name:              cm.cluster.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.cluster.MasterReservedIP == "auto")

	//cm.cluster.InstanceImage, err = cm.conn.getInstanceImage()
	//if err != nil {
	//	cm.cluster.StatusCause = err.Error()
	//	return errors.FromErr(err).WithContext(cm.ctx).Err()
	//}
	//cm.ctx.Logger().Infof("Image %v is using to create instance", cm.cluster.InstanceImage)

	err = cm.importPublicKey()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// ignore errors, since tags are simply informational.
	cm.createTags()

	err = cm.reserveIP()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// ------------------------Generate Certs and upload in Firebase @dipta----------------------

	if err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = cm.UploadStartupConfig(cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	cm.ctx.Logger().Info("Creating master instance")
	masterDroplet, err := im.createInstance(cm.cluster.KubernetesMasterName, api.RoleKubernetesMaster, cm.cluster.MasterSKU)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.conn.waitForInstance(masterDroplet.ID, "active"); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	im.applyTag(masterDroplet.ID)
	if cm.cluster.MasterReservedIP != "" {
		if err = im.assignReservedIP(cm.cluster.MasterReservedIP, masterDroplet.ID); err != nil {
			cm.cluster.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	masterInstance, err := im.newKubeInstance(masterDroplet.ID)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Role = api.RoleKubernetesMaster
	cm.cluster.MasterExternalIP = masterInstance.ExternalIP
	cm.cluster.MasterInternalIP = masterInstance.InternalIP
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.MasterExternalIP, "<><><><>", cm.cluster.MasterReservedIP)
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	//if err = cloud.GenClusterCerts(cm.ctx, cm.cluster); err != nil {
	//	cm.cluster.StatusCause = err.Error()
	//	return errors.FromErr(err).WithContext(cm.ctx).Err()
	//}
	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.cluster.DetectApiServerURL()
	// needed to get master_internal_ip
	if err = cm.cluster.Save(); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	//cm.UploadStartupConfig(cm.cluster)
	//
	//// reboot master to use cert with internal_ip as SANS
	//time.Sleep(60 * time.Second)

	//cm.ctx.Logger().Info("Rebooting master instance")
	//if err = im.reboot(masterDroplet.ID); err != nil {
	//	cm.cluster.StatusCause = err.Error()
	//	return errors.FromErr(err).WithContext(cm.ctx).Err()
	//}
	//cm.ctx.Logger().Info("Rebooted master instance")
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	// start nodes
	for _, ng := range req.NodeGroups {
		cm.ctx.Logger().Infof("Creating %v node with sku %v", ng.Count, ng.Sku)
		igm := &InstanceGroupManager{
			cm: cm,
			instance: cloud.Instance{
				Type: cloud.InstanceType{
					ContextVersion: cm.cluster.ContextVersion,
					Sku:            ng.Sku,

					Master:       false,
					SpotInstance: false,
				},
				Stats: cloud.GroupStats{
					Count: ng.Count,
				},
			},
			im: im,
		}
		err = igm.AdjustInstanceGroup()
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
	key, resp, err := cm.conn.client.Keys.Create(go_ctx.TODO(), &godo.KeyCreateRequest{
		Name:      cm.cluster.SSHKeyExternalID,
		PublicKey: string(cm.cluster.SSHKey.PublicKey),
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Debugln("DO response", resp, " errors", err)
	cm.ctx.Logger().Debugf("Created new ssh key with name=%v and id=%v", cm.cluster.SSHKeyExternalID, key.ID)
	cm.ctx.Logger().Info("SSH public key added")
	return nil
}

func (cm *clusterManager) createTags() error {
	tag := "KubernetesCluster:" + cm.cluster.Name
	_, _, err := cm.conn.client.Tags.Get(go_ctx.TODO(), tag)
	if err != nil {
		// Tag does not already exist
		_, _, err := cm.conn.client.Tags.Create(go_ctx.TODO(), &godo.TagCreateRequest{
			Name: tag,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Infof("Tag %v created", tag)
	}
	return nil
}

func (cm *clusterManager) reserveIP() error {
	if cm.cluster.MasterReservedIP == "auto" {
		fip, resp, err := cm.conn.client.FloatingIPs.Create(go_ctx.TODO(), &godo.FloatingIPCreateRequest{
			Region: cm.cluster.Region,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().Debugln("DO response", resp, " errors", err)
		cm.ctx.Logger().Infof("New floating ip %v reserved", fip.IP)
		cm.cluster.MasterReservedIP = fip.IP
	}
	return nil
}
