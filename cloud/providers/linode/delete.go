package linode

import (
	"fmt"
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) Delete(req *proto.ClusterDeleteRequest) error {
	defer cm.cluster.Delete()

	if cm.cluster.Status.Phase == api.ClusterPhasePending {
		cm.cluster.Status.Phase = api.ClusterPhaseFailing
	} else if cm.cluster.Status.Phase == api.ClusterPhaseReady {
		cm.cluster.Status.Phase = api.ClusterPhaseDeleting
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
	instances, err := cloud.Store(cm.ctx).Instances(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.cluster.Status.Reason != "" {
		errs = append(errs, cm.cluster.Status.Reason)
	}

	for _, i := range instances {
		backoff.Retry(func() error {
			linodeId, err := strconv.Atoi(i.Status.ExternalID)
			if err != nil {
				return err
			}
			_, err = cm.conn.client.Linode.Delete(linodeId, true)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		cloud.Logger(cm.ctx).Infof("Linode %v with id %v for clutser is deleted", i.Name, i.Status.ExternalID, cm.cluster.Name)
	}

	backoff.Retry(cm.deleteStackscripts, backoff.NewExponentialBackOff())
	cloud.Logger(cm.ctx).Infof("Stack scripts for cluster %v deleted", cm.cluster.Name)
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

	cloud.Logger(cm.ctx).Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) deleteStackscripts() error {
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

func (cm *ClusterManager) deleteSSHKey() (err error) {
	//if cm.cluster.Spec.SSHKeyPHID != "" {
	//	//updates := &storage.SSHKey{IsDeleted: 1}
	//	//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
	//	//_, err = cloud.Store(cm.ctx).Engine.Update(updates, cond)
	//	//cm.ctx.Notifier.StoreAndNotify(api.JobPhaseRunning, fmt.Sprintf("SSH key for cluster %v deleted", cm.ctx.MasterDiskId))
	//}
	return
}
