package packet

import (
	"fmt"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
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

	for _, i := range cm.ins.Instances {
		backoff.Retry(func() error {
			_, err := cm.conn.client.Devices.Delete(i.Status.ExternalID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("Droplet %v with id %v for clutser is deleted", i.Name, i.Status.ExternalID, cm.cluster.Name)
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
		if cm.cluster.Status.Phase == api.ClusterPhaseDeleting {
			cm.cluster.Status.Reason = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	cm.ctx.Logger().Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteSSHKey() (err error) {
	cm.ctx.Logger().Infof("Deleting SSH key for cluster", cm.cluster.Name)

	if cm.cluster.Spec.SSHKey != nil {
		backoff.Retry(func() error {
			_, err := cm.conn.client.SSHKeys.Delete(cm.cluster.Spec.SSHKeyExternalID)
			return err
		}, backoff.NewExponentialBackOff())
	}

	if cm.cluster.Spec.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store().Engine.Update(updates, cond)
	}
	return
}
