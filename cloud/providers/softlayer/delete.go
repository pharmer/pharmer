package softlayer

import (
	"fmt"
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
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
		cm.conn, err = NewConnector(cm.cluster)
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

	for _, i := range cm.ins.Instances {
		deviceID, _ := strconv.Atoi(i.ExternalID)
		backoff.Retry(func() error {
			err := cm.deleteInstance(deviceID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("Droplet %v with id %v for clutser is deleted", i.Name, i.ExternalID, cm.cluster.Name)
	}

	// Delete SSH key from DB
	if err := cm.deleteSSHKey(); err != nil {
		errs = append(errs, err.Error())
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

	cm.ctx.Logger().Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *clusterManager) deleteInstance(id int) error {
	service := cm.conn.virtualServiceClient.Id(id)
	success, err := service.DeleteObject()
	if err != nil {
		return errors.FromErr(err).Err()
	} else if !success {
		return errors.New("Error deleting virtual guest").Err()
	}
	return nil
}

func (cm *clusterManager) deleteSSHKey() (err error) {
	if cm.cluster.SSHKey != nil {
		sshid, _ := strconv.Atoi(cm.cluster.SSHKeyExternalID)
		backoff.Retry(func() error {
			service := cm.conn.securityServiceClient.Id(sshid)
			_, err := service.DeleteObject()
			return err
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("SSH key for cluster %v deleted", cm.cluster.Name)
	}

	if cm.cluster.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store().Engine.Update(updates, cond)
	}
	return
}