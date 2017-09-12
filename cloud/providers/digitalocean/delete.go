package digitalocean

import (
	gtx "context"
	"fmt"
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/errors"
	"github.com/appscode/pharmer/api"
	. "github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (cm *ClusterManager) Delete(req *proto.ClusterDeleteRequest) error {
	if cm.cluster.Status.Phase == api.ClusterPending {
		cm.cluster.Status.Phase = api.ClusterFailing
	} else if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	// Store(cm.ctx).UpdateKubernetesStatus(cm.ctx.PHID, cm.ctx.Status)

	var err error
	if cm.conn == nil {
		cm.conn, err = NewConnector(cm.ctx, cm.cluster)
		if err != nil {
			cm.cluster.Status.Reason = err.Error()
			return errors.FromErr(err).WithContext(cm.ctx).Err()
		}
	}
	cm.namer = namer{cluster: cm.cluster}
	instances, err := Store(cm.ctx).Instances(cm.cluster.Name).List(metav1.ListOptions{})
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
			dropletID, err := strconv.Atoi(i.Status.ExternalID)
			if err != nil {
				return err
			}
			_, err = cm.conn.client.Droplets.Delete(gtx.TODO(), dropletID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		Logger(cm.ctx).Infof("Droplet %v with id %v for clutser is deleted", i.Name, i.Status.ExternalID, cm.cluster.Name)
	}

	// delete by tag
	backoff.Retry(func() error {
		_, err := cm.conn.client.Droplets.DeleteByTag(gtx.TODO(), "KubernetesCluster:"+cm.cluster.Name)
		return err
	}, backoff.NewExponentialBackOff())
	Logger(cm.ctx).Infof("Deleted droplet by tag %v", "KubernetesCluster:"+cm.cluster.Name)

	if req.ReleaseReservedIp && cm.cluster.Spec.MasterReservedIP != "" {
		backoff.Retry(func() error {
			return cm.releaseReservedIP(cm.cluster.Spec.MasterReservedIP)
		}, backoff.NewExponentialBackOff())
	}

	// Delete SSH key from DB
	if err := cm.deleteSSHKey(); err != nil {
		errs = append(errs, err.Error())
	}

	if err := DeleteARecords(cm.ctx, cm.cluster); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.cluster.Status.Phase == api.ClusterDeleting {
			cm.cluster.Status.Reason = strings.Join(errs, "\n")
		}
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	Logger(cm.ctx).Infof("Cluster %v deletion is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) releaseReservedIP(ip string) error {
	resp, err := cm.conn.client.FloatingIPs.Delete(gtx.TODO(), ip)
	Logger(cm.ctx).Debugln("DO response", resp, " errors", err)
	if err != nil {
		return errors.FromErr(err).Err()
	}
	Logger(cm.ctx).Infof("Floating ip %v deleted", ip)
	return nil
}

func (cm *ClusterManager) deleteSSHKey() error {
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		_, err := cm.conn.client.Keys.DeleteByFingerprint(gtx.TODO(), SSHKey(cm.ctx).OpensshFingerprint)
		return err == nil, nil
	})
	if err != nil {
		return err
	}
	Logger(cm.ctx).Infof("SSH key for cluster %v deleted", cm.cluster.Name)

	//if cm.ctx.SSHKeyPHID != "" {
	//	//updates := &storage.SSHKey{IsDeleted: 1}
	//	//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
	//	// _, err = Store(cm.ctx).Engine.Update(updates, cond)
	//}
	return nil
}

func (cm *ClusterManager) deleteDroplet(dropletID int, nodeName string) error {
	_, err := cm.conn.client.Droplets.Delete(gtx.TODO(), dropletID)
	if err != nil {
		return err
	}
	Logger(cm.ctx).Infof("Droplet %v deleted", dropletID)
	return nil
}

func (cm *ClusterManager) deleteMaster(dropletID int) error {
	_, err := cm.conn.client.Droplets.Delete(gtx.TODO(), dropletID)
	if err != nil {
		return err
	}
	// TODO; FixIt!
	//for i, v := range instances {
	//	droplet, _ := strconv.Atoi(v.Status.ExternalID)
	//	if droplet == dropletID {
	//		cm.ins[i].Status.Phase = api.InstancePhaseDeleted
	//	}
	//}
	return err
}
