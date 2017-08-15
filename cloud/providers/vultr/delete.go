package vultr

import (
	"fmt"
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
	err = cm.ins.Load()
	if err != nil {
		cm.cluster.StatusCause = err.Error()
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}

	var errs []string
	if cm.cluster.StatusCause != "" {
		errs = append(errs, cm.cluster.StatusCause)
	}

	for _, i := range cm.ins.Instances {
		backoff.Retry(func() error {
			err := cm.conn.client.DeleteServer(i.ExternalID)
			if err != nil {
				return err
			}

			return nil
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("Instance %v with id %v for clutser is deleted", i.Name, i.ExternalID, cm.cluster.Name)
	}

	if req.ReleaseReservedIp && cm.cluster.MasterReservedIP != "" {
		backoff.Retry(func() error {
			return cm.releaseReservedIP(cm.cluster.MasterReservedIP)
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("Reserved ip for cluster %v", cm.cluster.Name)
	}

	cm.ctx.Logger().Infof("Deleting startup scripts for cluster %v", cm.cluster.Name)
	backoff.Retry(cm.deleteStartupScript, backoff.NewExponentialBackOff())

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

func (cm *clusterManager) releaseReservedIP(ip string) error {
	cm.ctx.Logger().Debugln("Deleting Floating IP", ip)
	err := cm.conn.client.DestroyReservedIP(ip)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *clusterManager) deleteStartupScript() error {
	scripts, err := cm.conn.client.GetStartupScripts()
	if err != nil {
		return err
	}
	for _, script := range scripts {
		if strings.HasPrefix(script.Name, cm.cluster.Name+"-") {
			err := cm.conn.client.DeleteStartupScript(script.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cm *clusterManager) deleteSSHKey() (err error) {
	cm.ctx.Logger().Infof("Deleting ssh key for cluster %v", cm.cluster.MasterDiskId)

	if cm.cluster.SSHKey != nil {
		backoff.Retry(func() error {
			return cm.conn.client.DeleteSSHKey(cm.cluster.SSHKeyExternalID)
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("SSH key for cluster %v is deleted", cm.cluster.Name)
	}

	if cm.cluster.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store().Engine.Update(updates, cond)
	}
	return
}
