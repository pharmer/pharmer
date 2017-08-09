package azure

import (
	"fmt"
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
	err := cm.deleteMaster()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}
	err = cm.UploadStartupConfig()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterNIC, err := im.conn.interfacesClient.Get(cm.namer.ResourceGroupName(), cm.namer.NetworkInterfaceName(cm.ctx.KubernetesMasterName), "")
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	as, err := im.getAvailablitySet()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	sa, err := im.getStorageAccount()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterScript := im.RenderStartupScript(cm.ctx.NewScriptOptions(), cm.ctx.MasterSKU, system.RoleKubernetesMaster)
	_, err = im.createVirtualMachine(masterNIC, as, sa, cm.namer.MasterName(), masterScript, cm.ctx.MasterSKU)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err := common.ProbeKubeAPI(cm.ctx); err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) deleteMaster() error {
	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}
	err := im.DeleteVirtualMachine(cm.namer.MasterName())
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	time.Sleep(1 * time.Minute)
	return nil
}

func (cm *clusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")

	im := &instanceManager{ctx: cm.ctx, conn: cm.conn, namer: cm.namer}

	igm := &InstanceGroupManager{cm: cm, im: im}
	oldinstances, err := igm.listInstances(sku)
	cm.UploadStartupConfig()
	//rolling update
	for _, instance := range oldinstances {
		err = im.DeleteVirtualMachine(instance.Name)
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.deleteNodeNetworkInterface(cm.namer.NetworkInterfaceName(instance.Name))
		if err != nil {
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		err = cm.deletePublicIp(cm.namer.PublicIPName(instance.Name))
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
