package azure

import (
	"fmt"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) SetVersion(req *proto.ClusterReconfigureRequest) error {
	//if !cloud.UpgradeRequired(cm.cluster, req) {
	//	cloud.Logger(cm.ctx).Infof("Upgrade command skipped for cluster %v", cm.cluster.Name)
	//	return nil
	//}
	if cm.conn == nil {
		conn, err := NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
		cm.conn = conn
	}

	cm.cluster.Generation = int64(0)
	cm.namer = namer{cluster: cm.cluster}
	// assign new timestamp and new launch_config version
	cm.cluster.Spec.KubernetesVersion = req.KubernetesVersion

	_, err := cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	fmt.Println("Updating...")
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
	_, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cloud.Logger(cm.ctx).Infof("Update Completed")
	return nil
}

func (cm *ClusterManager) updateMaster() error {
	err := cm.deleteMaster()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}
	masterNIC, err := im.conn.interfacesClient.Get(cm.namer.ResourceGroupName(), cm.namer.NetworkInterfaceName(cm.cluster.Spec.KubernetesMasterName), "")
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	as, err := im.getAvailablitySet()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	sa, err := im.getStorageAccount()
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	masterScript, err := cloud.RenderStartupScript(cm.ctx, cm.cluster, api.RoleMaster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	_, err = im.createVirtualMachine(masterNIC, as, sa, cm.namer.MasterName(), masterScript, cm.cluster.Spec.MasterSKU)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	if err := cloud.WaitForReadyMaster(cm.ctx, cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *ClusterManager) deleteMaster() error {
	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}
	err := im.DeleteVirtualMachine(cm.namer.MasterName())
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	time.Sleep(1 * time.Minute)
	return nil
}

func (cm *ClusterManager) updateNodes(sku string) error {
	fmt.Println("Updating Nodes...")

	im := &instanceManager{cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	igm := &NodeGroupManager{cm: cm, im: im}
	oldinstances, err := igm.listInstances(sku)
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
				ContextVersion: cm.cluster.Generation,
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
	}
	//currentIns, err := igm.listInstances(sku)
	//if err != nil {
	//	return errors.FromErr(err).WithContext(cm.ctx).Err()
	//}
	//err = cloud.AdjustDbInstance(cm.ctx, cm.ins, currentIns, sku)
	_, err = cloud.Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)

	return nil
}
