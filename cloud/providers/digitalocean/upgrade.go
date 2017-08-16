package digitalocean

import (
	"fmt"
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *clusterManager) setVersion(req *proto.ClusterReconfigureRequest) error {
	if !cloud.UpgradeRequired(cm.cluster, req) {
		cm.ctx.Logger().Infof("Upgrade command skipped for cluster %v", cm.cluster.Name)
		return nil
	}
	if cm.conn == nil {
		conn, err := NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}

	cm.cluster.ContextVersion = int64(0)
	cm.namer = namer{cluster: cm.cluster}
	// assign new timestamp and new launch_config version
	cm.cluster.EnvTimestamp = time.Now().UTC().Format("2006-01-02T15:04:05-0700")
	cm.cluster.KubeVersion = req.Version

	err := cm.ctx.Store().Clusters().SaveCluster(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	fmt.Println("Updating...")
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances, _ = cm.ctx.Store().Instances().LoadInstances(cm.cluster.Name)
	if req.ApplyToMaster {
		err = cm.updateMaster()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	} else {
		err = cm.updateNodes(req.Sku)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	err = cm.ctx.Store().Clusters().SaveCluster(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.ctx.Store().Instances().SaveInstances(cm.ins.Instances)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Update Completed")
	return nil
}

func (cm *clusterManager) updateMaster() error {
	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}
	masterInstanceID, err := im.getInstanceId(cm.namer.MasterName())
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cm.deleteMaster(masterInstanceID)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterDroplet, err := im.createInstance(cm.cluster.KubernetesMasterName, api.RoleKubernetesMaster, cm.cluster.MasterSKU)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.conn.waitForInstance(masterDroplet.ID, "active"); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	im.applyTag(masterDroplet.ID)
	time.Sleep(1 * time.Minute)
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
	fmt.Println("Master EXTERNAL IP ================", cm.cluster.MasterExternalIP, "<><><>", cm.cluster.MasterReservedIP)
	cm.ctx.Logger().Infof("Rebooting master instance")
	err = cloud.EnsureARecord(cm.ctx, cm.cluster, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cm.UploadStartupConfig(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = im.reboot(masterDroplet.ID); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	if err := cloud.ProbeKubeAPI(cm.ctx, cm.cluster); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")

	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	igm := &InstanceGroupManager{cm: cm, im: im}
	oldinstances, err := igm.listInstances(sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.UploadStartupConfig(cm.cluster)

	for _, instance := range oldinstances {
		dropletID, err := strconv.Atoi(instance.ExternalID)
		err = cm.deleteDroplet(dropletID, instance.InternalIP)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		igm.instance = cloud.Instance{
			Type: cloud.InstanceType{
				ContextVersion: cm.cluster.ContextVersion,
				Sku:            sku,
				Master:         false,
				SpotInstance:   false,
			},
		}
		_, err = igm.StartNode()
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}

		fmt.Println("Waiting for 1 minute")
		time.Sleep(1 * time.Minute)
		err = cloud.WaitForReadyNodes(cm.ctx, cm.cluster)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	currentIns, err := igm.listInstances(sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cloud.AdjustDbInstance(cm.ctx, cm.ins, currentIns, sku)
	// cluster.ctx.Instances = append(cluster.ctx.Instances, instances...)
	err = cm.ctx.Store().Clusters().SaveCluster(cm.cluster)

	return nil
}
