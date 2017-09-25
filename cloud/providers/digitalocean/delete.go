package digitalocean

import (
	gtx "context"

	"github.com/appscode/go/errors"
	. "github.com/appscode/pharmer/cloud"
	"k8s.io/apimachinery/pkg/util/wait"
)

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

func (cm *ClusterManager) deleteDroplet(dropletID int) error {
	_, err := cm.conn.client.Droplets.Delete(gtx.TODO(), dropletID)
	if err != nil {
		return err
	}
	Logger(cm.ctx).Infof("Droplet %v deleted", dropletID)
	return nil
}
