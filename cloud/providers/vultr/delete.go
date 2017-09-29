package vultr

import (
	"fmt"
	"strings"

	"github.com/appscode/go/errors"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	. "github.com/appscode/pharmer/cloud"
	"github.com/cenkalti/backoff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (cm *ClusterManager) Delete(rt api.RunType) (acts []api.Action, err error) {
	acts = make([]api.Action, 0)
	var instances []*api.NodeGroup
	instances, err = Store(cm.ctx).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.FromErr(err).WithContext(cm.ctx).Err()
		return
	}

	var errs []string
	if cm.cluster.Status.Reason != "" {
		errs = append(errs, cm.cluster.Status.Reason)
	}
	im := &instanceManager{ctx: cm.ctx, cluster: cm.cluster, conn: cm.conn, namer: cm.namer}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Instances",
		Message:  fmt.Sprintf("Instances of cluster %v will be deleted", cm.cluster.Name),
	})
	if rt != api.DryRun {
		for _, node := range instances {
			if node.IsMaster() {
				var id string
				if id, err = im.getInstance(cm.cluster.Spec.KubernetesMasterName); err != nil {
					return
				}

				backoff.Retry(func() error {
					err := cm.conn.client.DeleteServer(id)
					if err != nil {
						return err
					}

					return nil
				}, backoff.NewExponentialBackOff())
			}
			igm := &NodeGroupManager{
				cm: cm,
				instance: Instance{
					Type: InstanceType{
						Sku:          node.Spec.Template.Spec.SKU,
						Master:       false,
						SpotInstance: false,
					},
					Stats: GroupStats{
						Count: node.Spec.Nodes,
					},
				},
				im: im,
			}
			igm.deleteNodeGroup(node.Spec.Template.Spec.SKU)

			//Logger(cm.ctx).Infof("Instance %v with id %v for clutser is deleted", i.Name, i.Status.ExternalID, cm.cluster.Name)
		}
	}

	if cm.cluster.Spec.MasterReservedIP != "" {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Reserve IP",
			Message:  fmt.Sprintf("Reserved ip will be released"),
		})
		if rt != api.DryRun {
			backoff.Retry(func() error {
				return cm.releaseReservedIP(cm.cluster.Spec.MasterReservedIP)
			}, backoff.NewExponentialBackOff())
			Logger(cm.ctx).Infof("Reserved ip for cluster %v", cm.cluster.Name)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Startup script",
		Message:  fmt.Sprintf("Startup script of cluster %v will be deleted", cm.cluster.Name),
	})
	if rt != api.DryRun {
		Logger(cm.ctx).Infof("Deleting startup scripts for cluster %v", cm.cluster.Name)
		backoff.Retry(cm.deleteStartupScript, backoff.NewExponentialBackOff())
	}
	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "SSH key",
		Message:  fmt.Sprintf("SSH key will be deleted", cm.cluster.Name),
	})
	if rt != api.DryRun {
		// Delete SSH key from DB
		_, sshKeyID, er := im.getPublicKey()
		if er != nil {
			errs = append(errs, er.Error())
			return
		}
		if err := cm.deleteSSHKey(sshKeyID); err != nil {
			errs = append(errs, err.Error())
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "ARecord",
		Message:  fmt.Sprintf("ARecord will be deleted", cm.cluster.Name),
	})
	if rt != api.DryRun {
		if err := DeleteARecords(cm.ctx, cm.cluster); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		// Preserve statusCause for failed cluster
		if cm.cluster.Status.Phase == api.ClusterDeleting {
			cm.cluster.Status.Reason = strings.Join(errs, "\n")
		}
		err = fmt.Errorf(strings.Join(errs, "\n"))
		return
	}
	if rt != api.DryRun {
		cm.cluster.Status.Phase = api.ClusterDeleted
		Logger(cm.ctx).Infof("Cluster %v is deleted successfully", cm.cluster.Name)
	}
	return
}

func (cm *ClusterManager) releaseReservedIP(ip string) error {
	Logger(cm.ctx).Debugln("Deleting Floating IP", ip)
	err := cm.conn.client.DestroyReservedIP(ip)
	if err != nil {
		return errors.FromErr(err).WithContext(cm.ctx).Err()
	}
	return nil
}

func (cm *ClusterManager) deleteStartupScript() error {
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

func (cm *ClusterManager) deleteSSHKey(id string) error {
	Logger(cm.ctx).Infof("Deleting SSH key for cluster", cm.cluster.Name)
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		err := cm.conn.client.DeleteSSHKey(id)
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
