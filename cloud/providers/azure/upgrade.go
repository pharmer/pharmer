package azure

import (
	"fmt"
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
	err := cm.deleteMaster()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}
	err = cm.UploadStartupConfig()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterNIC, err := im.conn.interfacesClient.Get(cm.namer.ResourceGroupName(), cm.namer.NetworkInterfaceName(cm.cluster.KubernetesMasterName), "")
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	as, err := im.getAvailablitySet()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	sa, err := im.getStorageAccount()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterScript := im.RenderStartupScript(cm.cluster.MasterSKU, api.RoleKubernetesMaster)
	_, err = im.createVirtualMachine(masterNIC, as, sa, cm.namer.MasterName(), masterScript, cm.cluster.MasterSKU)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err := cloud.ProbeKubeAPI(cm.ctx, cm.cluster); err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) deleteMaster() error {
	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}
	err := im.DeleteVirtualMachine(cm.namer.MasterName())
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	time.Sleep(1 * time.Minute)
	return nil
}

func (cm *clusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")

	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

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
