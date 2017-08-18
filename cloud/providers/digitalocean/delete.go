package digitalocean

import (
	go_ctx "context"
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
			dropletID, err := strconv.Atoi(i.Status.ExternalID)
			if err != nil {
				return err
			}
			_, err = cm.conn.client.Droplets.Delete(go_ctx.TODO(), dropletID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("Droplet %v with id %v for clutser is deleted", i.Name, i.Status.ExternalID, cm.cluster.Name)
	}

	// delete by tag
	backoff.Retry(func() error {
		_, err := cm.conn.client.Droplets.DeleteByTag(go_ctx.TODO(), "KubernetesCluster:"+cm.cluster.Name)
		return err
	}, backoff.NewExponentialBackOff())
	cm.ctx.Logger().Infof("Deleted droplet by tag %v", "KubernetesCluster:"+cm.cluster.Name)

	if req.ReleaseReservedIp && cm.cluster.Spec.MasterReservedIP != "" {
		backoff.Retry(func() error {
			return cm.releaseReservedIP(cm.cluster.Spec.MasterReservedIP)
		}, backoff.NewExponentialBackOff())
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

	cm.ctx.Logger().Infof("Cluster %v deletion is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *clusterManager) releaseReservedIP(ip string) error {
	resp, err := cm.conn.client.FloatingIPs.Delete(go_ctx.TODO(), ip)
	cm.ctx.Logger().Debugln("DO response", resp, " errors", err)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	cm.ctx.Logger().Infof("Floating ip %v deleted", ip)
	return nil
}

func (cm *clusterManager) deleteSSHKey() (err error) {
	if cm.cluster.Spec.SSHKey != nil {
		backoff.Retry(func() error {
			_, err := cm.conn.client.Keys.DeleteByFingerprint(go_ctx.TODO(), cm.cluster.Spec.SSHKey.OpensshFingerprint)
			return err
		}, backoff.NewExponentialBackOff())
		cm.ctx.Logger().Infof("SSH key for cluster %v deleted", cm.cluster.Name)
	}

	//if cm.ctx.SSHKeyPHID != "" {
	//	//updates := &storage.SSHKey{IsDeleted: 1}
	//	//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
	//	// _, err = cm.ctx.Store().Engine.Update(updates, cond)
	//}
	return
}

func (cm *clusterManager) deleteDroplet(dropletID int, nodeName string) error {
	_, err := cm.conn.client.Droplets.Delete(go_ctx.TODO(), dropletID)
	if err != nil {
		return err
	}
	cm.ctx.Logger().Infof("Droplet %v deleted", dropletID)
	err = cloud.DeleteNodeApiCall(cm.ctx, cm.cluster, nodeName)
	if err != nil {
		return err
	}
	return nil
}

func (cm *clusterManager) deleteMaster(dropletID int) error {
	_, err := cm.conn.client.Droplets.Delete(go_ctx.TODO(), dropletID)
	if err != nil {
		return err
	}
	for i, v := range cm.ins.Instances {
		droplet, _ := strconv.Atoi(v.Status.ExternalID)
		if droplet == dropletID {
			cm.ins.Instances[i].Status.Phase = api.InstancePhaseDeleted
		}
	}
	return err
}
