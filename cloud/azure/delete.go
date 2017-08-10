package azure

import (
	"fmt"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/errorhandlers"
	"github.com/appscode/pharmer/storage"
)

func (cm *clusterManager) delete(req *proto.ClusterDeleteRequest) error {
	defer cm.ctx.Delete()

	if cm.ctx.Status == storage.KubernetesStatus_Pending {
		cm.ctx.Status = storage.KubernetesStatus_Failing
	} else if cm.ctx.Status == storage.KubernetesStatus_Ready {
		cm.ctx.Status = storage.KubernetesStatus_Deleting
	}
	// cm.ctx.Store.UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

	var err error
	if cm.conn == nil {
		cm.conn, err = NewConnector(cm.ctx)
		if err != nil {
			cm.ctx.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.namer = namer{ctx: cm.ctx}
	cm.ins, err = lib.NewInstances(cm.ctx)
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	err = cm.ins.Load()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.ctx.StatusCause != "" {
		errs = append(errs, cm.ctx.StatusCause)
	}

	cm.deleteResourceGroup(req.Name)
	for i := range cm.ins.Instances {
		cm.ins.Instances[i].Status = storage.KubernetesStatus_Deleted
	}
	if err := lib.DeleteARecords(cm.ctx); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.ctx.Status == storage.KubernetesStatus_Deleting {
			cm.ctx.StatusCause = strings.Join(errs, "\n")
		}
		errorhandlers.SendMailWithContextAndIgnore(cm.ctx, fmt.Errorf(strings.Join(errs, "\n")))
	}
	err = cm.ins.Save()
	if err != nil {
		cm.ctx.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Logger.Infof("Cluster %v is deleted successfully", cm.ctx.Name)
	return nil
}

func (cm *clusterManager) deleteResourceGroup(groupName string) error {
	_, err := cm.conn.groupsClient.Delete(groupName, make(chan struct{}))
	cm.ctx.Logger.Infof("Resource group %v deleted", groupName)
	return err
}

func (cm *clusterManager) deleteNodeNetworkInterface(interfaceName string) error {
	_, err := cm.conn.interfacesClient.Delete(cm.ctx.Name, interfaceName, make(chan struct{}))
	cm.ctx.Logger.Infof("Node network interface %v deleted", interfaceName)
	return err
}

func (cm *clusterManager) deletePublicIp(ipName string) error {
	_, err := cm.conn.publicIPAddressesClient.Delete(cm.ctx.Name, ipName, nil)
	cm.ctx.Logger.Infof("Public ip %v deleted", ipName)
	return err
}
func (cm *clusterManager) deleteVirtualMachine(machineName string) error {
	_, err := cm.conn.vmClient.Delete(cm.ctx.Name, machineName, make(chan struct{}))
	cm.ctx.Logger.Infof("Virtual machine %v deleted", machineName)
	return err
}
