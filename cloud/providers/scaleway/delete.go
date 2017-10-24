package scaleway

import (
	"fmt"
	"strings"

	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (cm *ClusterManager) Delete(req *api.ClusterDeleteRequest) error {
	if cm.cluster.Status.Phase == api.ClusterReady {
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
			err := cm.conn.client.DeleteServerForce(i.Status.ExternalID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		Logger(cm.ctx).Infof("Droplet %v with id %v for clutser is deleted", i.Name, i.Status.ExternalID, cm.cluster.Name)
	}

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

	Logger(cm.ctx).Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	return nil
}

func (cm *ClusterManager) releaseReservedIP(ip string) error {
	ips, err := cm.conn.client.GetIPS()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	for _, i := range ips.IPS {
		if i.Address == ip && i.Server == nil {
			err = cm.conn.client.DeleteIP(ip)
			if err != nil {
				return errors.FromErr(err).Err()
			}
		}
	}
	Logger(cm.ctx).Infof("Floating ip %v deleted", ip)
	return nil
}

func (cm *ClusterManager) deleteSSHKey() error {
	Logger(cm.ctx).Infof("Deleting SSH key for cluster", cm.cluster.Name)
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		user, err := cm.conn.client.GetUser()
		if err != nil {
			return err == nil, nil
		}
		sshPubKeys := make([]sapi.ScalewayKeyDefinition, 0)
		for _, k := range user.SSHPublicKeys {
			if k.Fingerprint != SSHKey(cm.ctx).OpensshFingerprint {
				sshPubKeys = append(sshPubKeys, sapi.ScalewayKeyDefinition{Key: k.Key})
			}
		}
		err = cm.conn.client.PatchUserSSHKey(user.ID, sapi.ScalewayUserPatchSSHKeyDefinition{
			SSHPublicKeys: sshPubKeys,
		})
		return err == nil, nil
	})
	if err != nil {
		return err
	}
	Logger(cm.ctx).Infof("SSH key for cluster %v deleted", cm.cluster.Name)

	//if cm.cluster.Spec.SSHKeyPHID != "" {
	//	//updates := &storage.SSHKey{IsDeleted: 1}
	//	//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
	//	//_, err = Store(cm.ctx).Engine.Update(updates, cond)
	//}
	return nil
}
