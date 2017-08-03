package softlayer

import (
	"fmt"
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/common"
	"github.com/appscode/pharmer/errorhandlers"
	"github.com/appscode/pharmer/storage"
	"github.com/cenkalti/backoff"
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
	cm.ins, err = common.NewInstances(cm.ctx)
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

	for _, i := range cm.ins.Instances {
		deviceID, _ := strconv.Atoi(i.ExternalID)
		backoff.Retry(func() error {
			err := cm.deleteInstance(deviceID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Droplet %v with id %v for clutser is deleted", i.Name, i.ExternalID, cm.ctx.Name))
	}

	// Delete SSH key from DB
	if err := cm.deleteSSHKey(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := common.DeleteARecords(cm.ctx); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.ctx.Status == storage.KubernetesStatus_Deleting {
			cm.ctx.StatusCause = strings.Join(errs, "\n")
		}
		errorhandlers.SendMailWithContextAndIgnore(cm.ctx, fmt.Errorf(strings.Join(errs, "\n")))
	}

	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Cluster %v is deleted successfully", cm.ctx.Name))
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
	if cm.ctx.SSHKey != nil {
		sshid, _ := strconv.Atoi(cm.ctx.SSHKeyExternalID)
		backoff.Retry(func() error {
			service := cm.conn.securityServiceClient.Id(sshid)
			_, err := service.DeleteObject()
			return err
		}, backoff.NewExponentialBackOff())
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("SSH key for cluster %v deleted", cm.ctx.Name))
	}

	if cm.ctx.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store.Engine.Update(updates, cond)
	}
	return
}
