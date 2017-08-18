package linode

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

	if cm.cluster.Status.Phase == api.KubernetesStatus_Pending {
		cm.cluster.Status.Phase = api.KubernetesStatus_Failing
	} else if cm.cluster.Status.Phase == api.KubernetesStatus_Ready {
		cm.cluster.Status.Phase = api.KubernetesStatus_Deleting
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
	cm.ins.Instances, err = cm.ctx.Store().Instances().LoadInstances(cm.cluster.Name)
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.cluster.Status.Reason != "" {
		errs = append(errs, cm.cluster.Status.Reason)
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
		cm.ctx.Logger().Infof("Linode %v with id %v for clutser is deleted", i.Name, i.ExternalID, cm.cluster.Name)
	}

	backoff.Retry(cm.deleteStackscripts, backoff.NewExponentialBackOff())
	cm.ctx.Logger().Infof("Stack scripts for cluster %v deleted", cm.cluster.Name)
	// Delete SSH key from DB
	if err := cm.deleteSSHKey(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := cloud.DeleteARecords(cm.ctx, cm.cluster); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.cluster.Status.Phase == api.KubernetesStatus_Deleting {
			cm.cluster.Status.Reason = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	cm.ctx.Logger().Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *clusterManager) deleteStackscripts() error {
	scripts, err := cm.conn.client.StackScript.List(0)
	if err != nil {
		return err
	}
	for _, script := range scripts.StackScripts {
		if strings.HasPrefix(script.Label.String(), cm.cluster.Name+"-") {
			_, err := cm.conn.client.StackScript.Delete(script.StackScriptId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cm *clusterManager) deleteSSHKey() (err error) {
	if cm.cluster.Spec.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store().Engine.Update(updates, cond)
		//cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("SSH key for cluster %v deleted", cm.ctx.MasterDiskId))
	}
	return
}
