package linode

import (
	"fmt"
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/cloud/lib"
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
	// cm.ctx.Store().UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

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

	for _, i := range cm.ins.Instances {
		backoff.Retry(func() error {
			linodeId, err := strconv.Atoi(i.ExternalID)
			if err != nil {
				return err
			}
			_, err = cm.conn.client.Linode.Delete(linodeId, true)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("Linode %v with id %v for clutser is deleted", i.Name, i.ExternalID, cm.ctx.Name)
	}

	backoff.Retry(cm.deleteStackscripts, backoff.NewExponentialBackOff())
	cm.ctx.Logger().Infof("Stack scripts for cluster %v deleted", cm.ctx.Name)
	// Delete SSH key from DB
	if err := cm.deleteSSHKey(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := lib.DeleteARecords(cm.ctx); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.ctx.Status == storage.KubernetesStatus_Deleting {
			cm.ctx.StatusCause = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	cm.ctx.Logger().Infof("Cluster %v is deleted successfully", cm.ctx.Name)
	return nil
}

func (cm *clusterManager) deleteStackscripts() error {
	scripts, err := cm.conn.client.StackScript.List(0)
	if err != nil {
		return err
	}
	for _, script := range scripts.StackScripts {
		if strings.HasPrefix(script.Label.String(), cm.ctx.Name+"-") {
			_, err := cm.conn.client.StackScript.Delete(script.StackScriptId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cm *clusterManager) deleteSSHKey() (err error) {
	if cm.ctx.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store().Engine.Update(updates, cond)
		//cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("SSH key for cluster %v deleted", cm.ctx.MasterDiskId))
	}
	return
}
