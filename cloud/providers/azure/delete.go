package azure

import (
	"fmt"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
)

func (cm *ClusterManager) Delete(req *proto.ClusterDeleteRequest) error {
	defer cm.cluster.Delete()

	if cm.cluster.Status.Phase == api.ClusterPending {
		cm.cluster.Status.Phase = api.ClusterFailing
	} else if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	// cloud.Store(cm.ctx).UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

	var err error
	if cm.conn == nil {
		cm.conn, err = NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.namer = namer{cluster: cm.cluster}
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.cluster.Status.Reason != "" {
		errs = append(errs, cm.cluster.Status.Reason)
	}

	cm.deleteResourceGroup(req.Name)
	//for i := range cm.ins.Instances {
	//	cm.ins.Instances[i].Status.Phase = api.ClusterPhaseDeleted
	//}
	if err := cloud.DeleteARecords(cm.ctx, cm.cluster); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.cluster.Status.Phase == api.ClusterDeleting {
			cm.cluster.Status.Reason = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	cloud.Logger(cm.ctx).Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteResourceGroup(groupName string) error {
	_, errchan := cm.conn.groupsClient.Delete(groupName, make(chan struct{}))
	cloud.Logger(cm.ctx).Infof("Resource group %v deleted", groupName)
	return <-errchan
}

func (cm *ClusterManager) deleteNodeNetworkInterface(interfaceName string) error {
	_, errchan := cm.conn.interfacesClient.Delete(cm.cluster.Name, interfaceName, make(chan struct{}))
	cloud.Logger(cm.ctx).Infof("Node network interface %v deleted", interfaceName)
	return <-errchan
}

func (cm *ClusterManager) deletePublicIp(ipName string) error {
	_, errchan := cm.conn.publicIPAddressesClient.Delete(cm.cluster.Name, ipName, nil)
	cloud.Logger(cm.ctx).Infof("Public ip %v deleted", ipName)
	return <-errchan
}
func (cm *ClusterManager) deleteVirtualMachine(machineName string) error {
	_, errchan := cm.conn.vmClient.Delete(cm.cluster.Name, machineName, make(chan struct{}))
	cloud.Logger(cm.ctx).Infof("Virtual machine %v deleted", machineName)
	return <-errchan
}
