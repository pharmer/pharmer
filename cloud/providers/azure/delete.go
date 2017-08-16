package azure

import (
	"fmt"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *clusterManager) delete(req *proto.ClusterDeleteRequest) error {
	defer cm.cluster.Delete()

	if cm.cluster.Status == api.KubernetesStatus_Pending {
		cm.cluster.Status = api.KubernetesStatus_Failing
	} else if cm.cluster.Status == api.KubernetesStatus_Ready {
		cm.cluster.Status = api.KubernetesStatus_Deleting
	}
	// cm.ctx.Store().UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

	var err error
	if cm.conn == nil {
		cm.conn, err = NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.StatusCause = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.namer = namer{cluster: cm.cluster}
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances, err = cm.ctx.Store().Instances().LoadInstances(cm.cluster.Name)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.cluster.StatusCause != "" {
		errs = append(errs, cm.cluster.StatusCause)
	}

	cm.deleteResourceGroup(req.Name)
	for i := range cm.ins.Instances {
		cm.ins.Instances[i].Status = api.KubernetesStatus_Deleted
	}
	if err := cloud.DeleteARecords(cm.ctx, cm.cluster); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.cluster.Status == api.KubernetesStatus_Deleting {
			cm.cluster.StatusCause = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	err = cm.ctx.Store().Instances().SaveInstances(cm.ins.Instances)
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Logger().Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *clusterManager) deleteResourceGroup(groupName string) error {
	_, err := cm.conn.groupsClient.Delete(groupName, make(chan struct{}))
	cm.ctx.Logger().Infof("Resource group %v deleted", groupName)
	return err
}

func (cm *clusterManager) deleteNodeNetworkInterface(interfaceName string) error {
	_, err := cm.conn.interfacesClient.Delete(cm.cluster.Name, interfaceName, make(chan struct{}))
	cm.ctx.Logger().Infof("Node network interface %v deleted", interfaceName)
	return err
}

func (cm *clusterManager) deletePublicIp(ipName string) error {
	_, err := cm.conn.publicIPAddressesClient.Delete(cm.cluster.Name, ipName, nil)
	cm.ctx.Logger().Infof("Public ip %v deleted", ipName)
	return err
}
func (cm *clusterManager) deleteVirtualMachine(machineName string) error {
	_, err := cm.conn.vmClient.Delete(cm.cluster.Name, machineName, make(chan struct{}))
	cm.ctx.Logger().Infof("Virtual machine %v deleted", machineName)
	return err
}
