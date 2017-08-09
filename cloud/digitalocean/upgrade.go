package digitalocean

import (
	"fmt"
	"strconv"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/data"
	"github.com/appscode/errors"
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/common"
	"github.com/appscode/pharmer/system"
)

func (cm *clusterManager) setVersion(req *proto.ClusterReconfigureRequest) error {
	if !common.UpgradeRequired(cm.ctx, req) {
		cm.ctx.Logger().Warningf("Upgrade command skipped for cluster %v", cm.ctx.Name)
		return nil
	}
	if cm.conn == nil {
		conn, err := NewConnector(cm.ctx)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}

	cm.ctx.ContextVersion = int64(0)
	cm.namer = namer{ctx: cm.ctx}
	// assign new timestamp and new launch_config version
	cm.ctx.EnvTimestamp = time.Now().UTC().Format("2006-01-02T15:04:05-0700")
	if req.Version != "" {
		if v, err := data.LoadKubernetesVersion(cm.ctx.Provider, _env.FromHost().String(), req.Version); err == nil {
			cm.ctx.SaltbaseVersion = v.Apps[system.AppKubeSaltbase]
			cm.ctx.KubeServerVersion = v.Apps[system.AppKubeServer]
			cm.ctx.KubeStarterVersion = v.Apps[system.AppKubeStarter]
			cm.ctx.HostfactsVersion = v.Apps[system.AppHostfacts]
		}
	}
	cm.ctx.KubeVersion = req.Version
	cm.ctx.Apps[system.AppKubeSaltbase] = system.NewAppKubernetesSalt(cm.ctx.Provider, cm.ctx.Region, cm.ctx.SaltbaseVersion)
	cm.ctx.Apps[system.AppKubeServer] = system.NewAppKubernetesServer(cm.ctx.Provider, cm.ctx.Region, cm.ctx.KubeServerVersion)
	cm.ctx.Apps[system.AppKubeStarter] = system.NewAppStartKubernetes(cm.ctx.Provider, cm.ctx.Region, cm.ctx.KubeStarterVersion)

	err := cm.ctx.Save()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	fmt.Println("Updating...")
	cm.ins, err = common.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Load()
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
	err = cm.ctx.Save()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.ins.Save()
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ctx.Logger().Infof("Update Completed")
	return nil
}

func (cm *clusterManager) updateMaster() error {
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}
	masterInstanceID, err := im.getInstanceId(cm.namer.MasterName())
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cm.deleteMaster(masterInstanceID)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	masterDroplet, err := im.createInstance(cm.ctx.KubernetesMasterName, system.RoleKubernetesMaster, cm.ctx.MasterSKU)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err = cm.conn.waitForInstance(masterDroplet.ID, "active"); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	im.applyTag(masterDroplet.ID)
	time.Sleep(1 * time.Minute)
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
	fmt.Println("Master EXTERNAL IP ================", cm.ctx.MasterExternalIP, "<><><>", cm.ctx.MasterReservedIP)
	cm.ctx.Logger().Infof("Rebooting master instance")
	err = common.EnsureARecord(cm.ctx, masterInstance) // works for reserved or non-reserved mode
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	err = cm.UploadStartupConfig(cm.ctx)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	if err = im.reboot(masterDroplet.ID); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances = append(cm.ins.Instances, masterInstance)
	if err := common.ProbeKubeAPI(cm.ctx); err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")

	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}

	igm := &InstanceGroupManager{cm: cm, im: im}
	oldinstances, err := igm.listInstances(sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.UploadStartupConfig(cm.ctx)

	for _, instance := range oldinstances {
		dropletID, err := strconv.Atoi(instance.ExternalID)
		err = cm.deleteDroplet(dropletID, instance.InternalIP)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		igm.instance = common.Instance{
			Type: common.InstanceType{
				ContextVersion: cm.ctx.ContextVersion,
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
		err = common.WaitForReadyNodes(cm.ctx)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	currentIns, err := igm.listInstances(sku)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = common.AdjustDbInstance(cm.ins, currentIns, sku)
	// cluster.ctx.Instances = append(cluster.ctx.Instances, instances...)
	err = cm.ctx.Save()

	return nil
}
