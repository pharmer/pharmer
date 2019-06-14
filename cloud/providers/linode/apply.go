package linode

import (
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func (cm *ClusterManager) EnsureMaster(acts []api.Action, dryRun bool) ([]api.Action, error) {
	leaderMachine, err := GetLeaderMachine(cm.Cluster)
	if err != nil {
		return acts, err
	}

	acts = append(acts, api.Action{
		Action:   api.ActionAdd,
		Resource: "Master startup script",
		Message:  "Startup script will be created/updated for master instance",
	})

	if !dryRun {
		script, err := RenderStartupScript(cm, leaderMachine, "", customTemplate)
		if err != nil {
			return acts, err
		}

		if _, err = cm.conn.createOrUpdateStackScript(leaderMachine, script); err != nil {
			return acts, err
		}
	}

	if d, _ := cm.conn.instanceIfExists(leaderMachine); d == nil {
		log.Info("Creating master instance")
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Master instance %s will be created", leaderMachine.Name),
		})
		if !dryRun {
			var masterServer *api.NodeInfo
			nodeAddresses := make([]corev1.NodeAddress, 0)
			if cm.Cluster.Status.Cloud.LoadBalancer.DNS != "" {
				nodeAddresses = append(nodeAddresses, corev1.NodeAddress{
					Type:    corev1.NodeExternalDNS,
					Address: cm.Cluster.Status.Cloud.LoadBalancer.DNS,
				})
			} else if cm.Cluster.Status.Cloud.LoadBalancer.IP != "" {
				nodeAddresses = append(nodeAddresses, corev1.NodeAddress{
					Type:    corev1.NodeExternalIP,
					Address: cm.Cluster.Status.Cloud.LoadBalancer.IP,
				})
			}

			script, err := RenderStartupScript(cm, leaderMachine, "", customTemplate)
			if err != nil {
				return nil, err
			}

			masterServer, err = cm.conn.CreateInstance(leaderMachine, script)
			if err != nil {
				return acts, err
			}

			if err = cm.conn.addNodeToBalancer(cm.namer.LoadBalancerName(), leaderMachine.Name, masterServer.PrivateIP); err != nil {
				return acts, err
			}

			if err = cm.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
				return acts, err
			}
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "MasterInstance",
				Message:  "master instance(s) already exist",
			})
		}
		if _, err = store.StoreProvider.Clusters().Update(cm.Cluster); err != nil {
			return acts, err
		}
	}

	return acts, nil
}

func (cm *ClusterManager) PrepareCloud(dryRun bool) ([]api.Action, error) {
	var acts []api.Action

	lb, err := cm.conn.lbByName(cm.namer.LoadBalancerName())

	if err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Balancer",
			Message:  fmt.Sprintf("Load Balancer will be created"),
		})

		lb, err = cm.conn.createLoadBalancer(cm.namer.LoadBalancerName())
		if err != nil {
			return acts, err
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionAdd,
		Resource: "Load Balancer",
		Message:  fmt.Sprintf("Load Balancer %q found", *lb.Label),
	})

	cm.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   *lb.IPv4,
		Port: api.DefaultKubernetesBindPort,
	}

	nodeAddresses := []corev1.NodeAddress{
		{
			Type:    corev1.NodeExternalIP,
			Address: cm.Cluster.Status.Cloud.LoadBalancer.IP,
		},
	}

	if err = cm.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
		return acts, errors.Wrap(err, "Error setting controlplane endpoints")
	}

	return acts, err
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	return "g6-standard-2"
}

//Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) ApplyDelete(dryRun bool) (acts []api.Action, err error) {
	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}
	var nodeInstances *corev1.NodeList
	nodeInstances, err = kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.RoleNodeKey: "",
		}).String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		log.Infof("node instance not found. Reason: %v", err)
	} else if err == nil {
		for _, mi := range nodeInstances.Items {

			if err = cm.conn.DeleteStackScript(mi.Name, api.RoleNode); err != nil {
				log.Infof("Reason: %v", err)
			}
			err = kc.CoreV1().Nodes().Delete(mi.Name, nil)
			if err != nil {
				log.Infof("Failed to delete node %s. Reason: %s", mi.Name, err)
			}
		}
	}

	var masterInstances *corev1.NodeList
	masterInstances, err = kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.RoleMasterKey: "",
		}).String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		log.Infof("master instance not found. Reason: %v", err)
	} else if err == nil {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Will delete master instance with name %v-master", cm.Cluster.Name),
		})
		if !dryRun {
			for _, mi := range masterInstances.Items {
				if err = cm.conn.DeleteStackScript(mi.Name, api.RoleMaster); err != nil {
					log.Infof("Reason: %v", err)
				}

				err = cm.conn.DeleteInstanceByProviderID(mi.Spec.ProviderID)
				if err != nil {
					log.Infof("Failed to delete instance %s. Reason: %s", mi.Spec.ProviderID, err)
				}
			}

		}
	}

	_, err = cm.conn.lbByName(cm.namer.LoadBalancerName())
	if err == errLBNotFound {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Load Balancer",
			Message:  "Load Balancer not found",
		})
	} else {
		if !dryRun {
			if err = cm.conn.deleteLoadBalancer(cm.namer.LoadBalancerName()); err != nil {
				return
			}
		}

		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "Load Balancer",
			Message:  "Load Balancer deleted",
		})
	}

	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return
	}

	log.Infof("Cluster %v deletion is deleted successfully", cm.Cluster.Name)
	return
}
