package softlayer

import (
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
		deviceID, _ := strconv.Atoi(i.Status.ExternalID)
		backoff.Retry(func() error {
			err := cm.deleteInstance(deviceID)
			if err != nil {
				return err
			}
			return nil
		}, backoff.NewExponentialBackOff())
		Logger(cm.ctx).Infof("Droplet %v with id %v for clutser is deleted", i.Name, i.Status.ExternalID, cm.cluster.Name)
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

func (cm *ClusterManager) deleteInstance(id int) error {
	service := cm.conn.virtualServiceClient.Id(id)
	success, err := service.DeleteObject()
	if err != nil {
		return errors.FromErr(err).Err()
	} else if !success {
		return errors.New("Error deleting virtual guest").Err()
	}
	return nil
}

func (cm *ClusterManager) deleteSSHKey() error {
	Logger(cm.ctx).Infof("Deleting SSH key for cluster", cm.cluster.Name)
	err := wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		sshid, _ := strconv.Atoi(cm.cluster.Status.SSHKeyExternalID)
		_, err := cm.conn.securityServiceClient.Id(sshid).DeleteObject()
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
