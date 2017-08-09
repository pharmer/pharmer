package digitalocean

import (
	"context"
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/common"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/system"
	"github.com/digitalocean/godo"
)

func (cm *clusterManager) create(req *proto.ClusterCreateRequest) error {
	err := cm.initContext(req)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins, err = common.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.conn, err = NewConnector(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Save()

	defer func(releaseReservedIp bool) {
		if cm.ctx.Status == storage.KubernetesStatus_Pending {
			cm.ctx.Status = storage.KubernetesStatus_Failing
		}
		cm.ctx.Save()
		cm.ins.Save()
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Cluster %v is %v", cm.ctx.Name, cm.ctx.Status))
		if cm.ctx.Status != storage.KubernetesStatus_Ready {
			cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Cluster %v is deleting", cm.ctx.Name))
			cm.delete(&proto.ClusterDeleteRequest{
				Name:              cm.ctx.Name,
				ReleaseReservedIp: releaseReservedIp,
			})
		}
	}(cm.ctx.MasterReservedIP == "auto")

	cm.ctx.InstanceImage, err = cm.conn.getInstanceImage()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Image %v is using to create instance", cm.ctx.InstanceImage))

	err = cm.importPublicKey()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// ignore errors, since tags are simply informational.
	cm.createTags()

	err = cm.reserveIP()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// -------------------------------------------------------------------ASSETS
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}

	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Creating master instance")
	masterDroplet, err := im.createInstance(cm.ctx.KubernetesMasterName, system.RoleKubernetesMaster, cm.ctx.MasterSKU)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.conn.waitForInstance(masterDroplet.ID, "active"); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	im.applyTag(masterDroplet.ID)
	if cm.ctx.MasterReservedIP != "" {
		if err = im.assignReservedIP(cm.ctx.MasterReservedIP, masterDroplet.ID); err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	masterInstance, err := im.newKubeInstance(masterDroplet.ID)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterInstance.Role = system.RoleKubernetesMaster
	cm.ctx.MasterExternalIP = masterInstance.ExternalIP
	cm.ctx.MasterInternalIP = masterInstance.InternalIP
	fmt.Println("Master EXTERNAL IP ================", cm.ctx.MasterExternalIP, "<><><><>", cm.ctx.MasterReservedIP)
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)

	if err = common.GenClusterCerts(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = common.EnsureARecord(cm.ctx, masterInstance) // works for reserved or non-reserved mode
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
	cm.UploadStartupConfig(cm.ctx)

	// reboot master to use cert with internal_ip as SANS
	time.Sleep(60 * time.Second)

	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Rebooting master instance")
	if err = im.reboot(masterDroplet.ID); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Rebooted master instance")
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	// start nodes
	for _, ng := range req.NodeGroups {
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Creating %v node with sku %v", ng.Count, ng.Sku))
		igm := &InstanceGroupManager{
			cm: cm,
			instance: common.Instance{
				Type: common.InstanceType{
					ContextVersion: cm.ctx.ContextVersion,
					Sku:            ng.Sku,

					Master:       false,
					SpotInstance: false,
				},
				Stats: common.GroupStats{
					Count: ng.Count,
				},
			},
			im: im,
		}
		err = igm.AdjustInstanceGroup()
	}

	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "Waiting for cluster initialization")

	// Wait for master A record to propagate
	if err := common.EnsureDnsIPLookup(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	// wait for nodes to start
	if err := common.ProbeKubeAPI(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// check all components are ok
	if err = common.CheckComponentStatuses(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	// Make sure nodes are connected to master and are ready
	if err = common.WaitForReadyNodes(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Status = storage.KubernetesStatus_Ready
	return nil
}

func (cm *clusterManager) importPublicKey() error {
	key, resp, err := cm.conn.client.Keys.Create(context.TODO(), &godo.KeyCreateRequest{
		Name:      cm.ctx.SSHKeyExternalID,
		PublicKey: string(cm.ctx.SSHKey.PublicKey),
	})
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().V(6).Infoln("DO response", resp, " errors", err)
	cm.ctx.Logger().Debugf("Created new ssh key with name=%v and id=%v", cm.ctx.SSHKeyExternalID, key.ID)
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, "SSH public key added")
	return nil
}

func (cm *clusterManager) createTags() error {
	tag := "KubernetesCluster:" + cm.ctx.Name
	_, _, err := cm.conn.client.Tags.Get(context.TODO(), tag)
	if err != nil {
		// Tag does not already exist
		_, _, err := cm.conn.client.Tags.Create(context.TODO(), &godo.TagCreateRequest{
			Name: tag,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Tag %v created", tag))
	}
	return nil
}

func (cm *clusterManager) reserveIP() error {
	if cm.ctx.MasterReservedIP == "auto" {
		fip, resp, err := cm.conn.client.FloatingIPs.Create(context.TODO(), &godo.FloatingIPCreateRequest{
			Region: cm.ctx.Region,
		})
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.ctx.Logger().V(6).Infoln("DO response", resp, " errors", err)
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("New floating ip %v reserved", fip.IP))
		cm.ctx.MasterReservedIP = fip.IP
	}
	return nil
}
