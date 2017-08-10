package digitalocean

import (
	"context"
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
			dropletID, err := strconv.Atoi(i.ExternalID)
			if err != nil {
				return err
			}
			_, err = cm.conn.client.Droplets.Delete(context.TODO(), dropletID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger.Infof("Droplet %v with id %v for clutser is deleted", i.Name, i.ExternalID, cm.ctx.Name)
	}

	// delete by tag
	backoff.Retry(func() error {
		_, err := cm.conn.client.Droplets.DeleteByTag(context.TODO(), "KubernetesCluster:"+cm.ctx.Name)
		return err
	}, backoff.NewExponentialBackOff())
	cm.ctx.Logger.Infof("Deleted droplet by tag %v", "KubernetesCluster:"+cm.ctx.Name)

	if req.ReleaseReservedIp && cm.ctx.MasterReservedIP != "" {
		backoff.Retry(func() error {
			return cm.releaseReservedIP(cm.ctx.MasterReservedIP)
		}, backoff.NewExponentialBackOff())
	}

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

	cm.ctx.Logger.Infof("Cluster %v deletion is deleted successfully", cm.ctx.Name)
	return nil
}

func (cm *clusterManager) releaseReservedIP(ip string) error {
	resp, err := cm.conn.client.FloatingIPs.Delete(context.TODO(), ip)
	cm.ctx.Logger.Debugln("DO response", resp, " errors", err)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	cm.ctx.Logger.Infof("Floating ip %v deleted", ip)
	return nil
}

func (cm *clusterManager) deleteSSHKey() (err error) {
	if cm.ctx.SSHKey != nil {
		backoff.Retry(func() error {
			_, err := cm.conn.client.Keys.DeleteByFingerprint(context.TODO(), cm.ctx.SSHKey.OpensshFingerprint)
			return err
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger.Infof("SSH key for cluster %v deleted", cm.ctx.Name)
	}

	//if cm.ctx.SSHKeyPHID != "" {
	//	//updates := &storage.SSHKey{IsDeleted: 1}
	//	//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
	//	// _, err = cm.ctx.Store.Engine.Update(updates, cond)
	//}
	return
}

func (cm *clusterManager) deleteDroplet(dropletID int, nodeName string) error {
	_, err := cm.conn.client.Droplets.Delete(context.TODO(), dropletID)
	if err != nil {
		return err
	}
	cm.ctx.Logger.Infof("Droplet %v deleted", dropletID)
	err = lib.DeleteNodeApiCall(cm.ctx, nodeName)
	if err != nil {
		return err
	}
	return nil
}

func (cm *clusterManager) deleteMaster(dropletID int) error {
	_, err := cm.conn.client.Droplets.Delete(context.TODO(), dropletID)
	if err != nil {
		return err
	}
	for i, v := range cm.ins.Instances {
		droplet, _ := strconv.Atoi(v.ExternalID)
		if droplet == dropletID {
			cm.ins.Instances[i].Status = storage.KubernetesInstanceStatus_Deleted
		}
	}
	return err
}
