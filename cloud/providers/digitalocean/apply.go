package digitalocean

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func (cm *ClusterManager) EnsureMaster() error {
	leaderMachine, err := GetLeaderMachine(cm.Cluster)
	if err != nil {
		return err
	}

	if d, _ := cm.conn.instanceIfExists(leaderMachine); d == nil {
		log.Info("Creating master instance")
		nodeAddresses := make([]core.NodeAddress, 0)

		if cm.Cluster.Status.Cloud.LoadBalancer.DNS != "" {
			nodeAddresses = append(nodeAddresses, core.NodeAddress{
				Type:    core.NodeExternalDNS,
				Address: cm.Cluster.Status.Cloud.LoadBalancer.DNS,
			})
		} else if cm.Cluster.Status.Cloud.LoadBalancer.IP != "" {
			nodeAddresses = append(nodeAddresses, core.NodeAddress{
				Type:    core.NodeExternalIP,
				Address: cm.Cluster.Status.Cloud.LoadBalancer.IP,
			})
		}

		_, err = cm.conn.CreateInstance(cm.Cluster, leaderMachine, "")
		if err != nil {
			return err
		}

		if err = cm.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
			return err
		}

	}

	if cm.Cluster, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
		return err
	}

	return nil
}

func (cm *ClusterManager) PrepareCloud() error {
	var found bool
	var err error

	found, _, err = cm.conn.getPublicKey()
	if err != nil {
		return err
	}
	if !found {
		cm.Cluster.Status.Cloud.SShKeyExternalID, err = cm.conn.importPublicKey()
		if err != nil {
			return err
		}
	}

	// ignore errors, since tags are simply informational.
	found, err = cm.conn.getTags()
	if err != nil {
		return err
	}
	if !found {
		if err = cm.conn.createTags(); err != nil {
			return err
		}
	}

	lb, err := cm.conn.lbByName(context.Background(), cm.namer.LoadBalancerName())
	if err == errLBNotFound {
		lb, err = cm.conn.createLoadBalancer(context.Background(), cm.namer.LoadBalancerName())
		if err != nil {
			return err
		}
	}

	cm.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   lb.IP,
		Port: lb.ForwardingRules[0].EntryPort,
	}

	nodeAddresses := []corev1.NodeAddress{
		{
			Type:    corev1.NodeExternalIP,
			Address: cm.Cluster.Status.Cloud.LoadBalancer.IP,
		},
	}

	if err = cm.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
		return errors.Wrap(err, "Error setting controlplane endpoints")
	}

	return nil
}

// Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) ApplyDelete() error {
	log.Infoln("deleting cluster")
	var found bool

	if cm.Cluster.Status.Phase == api.ClusterReady {
		cm.Cluster.Status.Phase = api.ClusterDeleting
	}
	_, err := store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}

	err = DeleteAllWorkerMachines(cm)
	if err != nil {
		log.Infof("failed to delete nodes: %v", err)
	}

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return err
	}
	var masterInstances *core.NodeList
	masterInstances, err = kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.RoleMasterKey: "",
		}).String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		log.Infof("master instance not found. Reason: %v", err)
	} else if err == nil {
		for _, mi := range masterInstances.Items {
			err = cm.conn.DeleteInstanceByProviderID(mi.Spec.ProviderID)
			if err != nil {
				log.Infof("Failed to delete instance %s. Reason: %s", mi.Spec.ProviderID, err)
			}
		}
	}

	// delete by tag
	tag := "KubernetesCluster:" + cm.Cluster.Name
	_, err = cm.conn.client.Droplets.DeleteByTag(context.Background(), tag)
	if err != nil {
		log.Infof("Failed to delete resources by tag %s. Reason: %s", tag, err)
	}
	log.Infof("Deleted droplet by tag %s", tag)

	// Delete SSH key
	found, _, err = cm.conn.getPublicKey()
	if err != nil {
		return err
	}
	if found {
		err = cm.conn.deleteSSHKey()
		if err != nil {
			return err
		}
	}

	_, err = cm.conn.lbByName(context.Background(), cm.namer.LoadBalancerName())
	if err != errLBNotFound {
		if err = cm.conn.deleteLoadBalancer(context.Background(), cm.namer.LoadBalancerName()); err != nil {
			return err
		}

	}

	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}

	log.Infof("Cluster %v deletion is deleted successfully", cm.Cluster.Name)
	return err
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	panic("implement me")
}
