package hetzner

import (
	"fmt"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	hc "github.com/appscode/go-hetzner"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/common"
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
		_, _, err := cm.conn.client.Server.CancelServer(&hc.CancelServerRequest{
			ServerIP:         i.ExternalIP,
			CancellationDate: time.Now().Format("2006-01-02"),
		})
		if err != nil {
			return err
		}

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

func (cm *clusterManager) deleteSSHKey() (err error) {
	cm.ctx.Logger().Infof("Deleting SSH key for cluster", cm.ctx.Name)

	if cm.ctx.SSHKey != nil {
		_, err = cm.conn.client.SSHKey.Delete(cm.ctx.SSHKey.OpensshFingerprint)
	}

	if cm.ctx.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store.Engine.Update(updates, cond)
	}
	return
}
