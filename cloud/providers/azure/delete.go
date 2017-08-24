package azure

import (
	"fmt"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) Delete(req *proto.ClusterDeleteRequest) error {
	defer cm.cluster.Delete()

	if cm.cluster.Status.Phase == api.ClusterPhasePending {
		cm.cluster.Status.Phase = api.ClusterPhaseFailing
	} else if cm.cluster.Status.Phase == api.ClusterPhaseReady {
		cm.cluster.Status.Phase = api.ClusterPhaseDeleting
	}
	// cm.ctx.Store().UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

	var err error
	if cm.conn == nil {
		cm.conn, err = NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.namer = namer{cluster: cm.cluster}
	cm.ins, err = cloud.NewInstances(cm.ctx, cm.cluster)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	cm.ins.Instances, err = cm.ctx.Store().Instances(cm.cluster.Name).List(api.ListOptions{})
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.cluster.Status.Reason != "" {
		errs = append(errs, cm.cluster.Status.Reason)
	}

	cm.deleteResourceGroup(req.Name)
	for i := range cm.ins.Instances {
		cm.ins.Instances[i].Status.Phase = api.ClusterPhaseDeleted
	}
	if err := cloud.DeleteARecords(cm.ctx, cm.cluster); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.cluster.Status.Phase == api.ClusterPhaseDeleting {
			cm.cluster.Status.Reason = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	err = cm.ctx.Store().Instances(cm.cluster.Name).SaveInstances(cm.ins.Instances)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	cm.ctx.Logger().Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteResourceGroup(groupName string) error {
	_, errchan := cm.conn.groupsClient.Delete(groupName, make(chan struct{}))
	cm.ctx.Logger().Infof("Resource group %v deleted", groupName)
	return <-errchan
}

func (cm *ClusterManager) deleteNodeNetworkInterface(interfaceName string) error {
	_, errchan := cm.conn.interfacesClient.Delete(cm.cluster.Name, interfaceName, make(chan struct{}))
	cm.ctx.Logger().Infof("Node network interface %v deleted", interfaceName)
	return <-errchan
}

func (cm *ClusterManager) deletePublicIp(ipName string) error {
	_, errchan := cm.conn.publicIPAddressesClient.Delete(cm.cluster.Name, ipName, nil)
	cm.ctx.Logger().Infof("Public ip %v deleted", ipName)
	return <-errchan
}
func (cm *ClusterManager) deleteVirtualMachine(machineName string) error {
	_, errchan := cm.conn.vmClient.Delete(cm.cluster.Name, machineName, make(chan struct{}))
	cm.ctx.Logger().Infof("Virtual machine %v deleted", machineName)
	return <-errchan
}
