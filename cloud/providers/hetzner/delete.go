package hetzner

import (
	"fmt"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	hc "github.com/appscode/go-hetzner"
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

func (cm *clusterManager) deleteSSHKey() (err error) {
	cm.ctx.Logger().Infof("Deleting SSH key for cluster", cm.cluster.Name)

	if cm.cluster.SSHKey != nil {
		_, err = cm.conn.client.SSHKey.Delete(cm.cluster.SSHKey.OpensshFingerprint)
	}

	if cm.cluster.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store().Engine.Update(updates, cond)
	}
	return
}