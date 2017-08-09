package scaleway

import (
	"fmt"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/cloud/lib"
	"github.com/appscode/pharmer/errorhandlers"
	"github.com/appscode/pharmer/storage"
	"github.com/cenkalti/backoff"
	sapi "github.com/scaleway/scaleway-cli/pkg/api"
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
			err := cm.conn.client.DeleteServerForce(i.ExternalID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Droplet %v with id %v for clutser is deleted", i.Name, i.ExternalID, cm.ctx.Name))
	}

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
		errorhandlers.SendMailWithContextAndIgnore(cm.ctx, fmt.Errorf(strings.Join(errs, "\n")))
	}

	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Cluster %v is deleted successfully", cm.ctx.Name))
	return nil
}

func (cm *clusterManager) releaseReservedIP(ip string) error {
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
	cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("Floating ip %v deleted", ip))
	return nil
}

func (cm *clusterManager) deleteSSHKey() (err error) {
	if cm.ctx.SSHKey != nil {
		backoff.Retry(func() error {
			user, err := cm.conn.client.GetUser()
			if err != nil {
				return err
			}

			sshPubKeys := make([]sapi.ScalewayKeyDefinition, 0)
			for _, k := range user.SSHPublicKeys {
				if k.Fingerprint != cm.ctx.SSHKey.OpensshFingerprint {
					sshPubKeys = append(sshPubKeys, sapi.ScalewayKeyDefinition{Key: k.Key})
				}
			}

			return cm.conn.client.PatchUserSSHKey(user.ID, sapi.ScalewayUserPatchSSHKeyDefinition{
				SSHPublicKeys: sshPubKeys,
			})
		}, backoff.NewExponentialBackOff())
		cm.ctx.Notifier.StoreAndNotify(api.JobStatus_Running, fmt.Sprintf("SSH key for cluster %v deleted", cm.ctx.Name))
	}

	if cm.ctx.SSHKeyPHID != "" {
		//updates := &storage.SSHKey{IsDeleted: 1}
		//cond := &storage.SSHKey{PHID: cm.ctx.SSHKeyPHID}
		//_, err = cm.ctx.Store.Engine.Update(updates, cond)
	}
	return
}
