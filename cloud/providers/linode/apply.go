package linode

import (
	"encoding/json"
	"fmt"

	semver "github.com/appscode/go-version"
	"github.com/appscode/go/log"
	. "github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error
	var acts []api.Action

	if in.Status.Phase == "" {
		return nil, errors.Errorf("cluster `%s` is in unknown phase", cm.cluster.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.conn, err = PrepareCloud(cm.ctx, in.Name, cm.owner); err != nil {
		return nil, err
	}
	cm.ctx = cm.conn.ctx
	if cm.cluster.Spec.Config.Cloud.InstanceImage, err = cm.conn.DetectInstanceImage(); err != nil {
		return nil, err
	}
	Logger(cm.ctx).Debugln("Linode instance image", cm.cluster.Spec.Config.Cloud.InstanceImage)
	if cm.cluster.Spec.Config.Cloud.Linode.KernelId, err = cm.conn.DetectKernel(); err != nil {
		return nil, err
	}
	Logger(cm.ctx).Infof("Linode kernel %v found", cm.cluster.Spec.Config.Cloud.Linode.KernelId)

	if cm.cluster.Status.Phase == api.ClusterUpgrading {
		return nil, errors.Errorf("cluster `%s` is upgrading. Retry after cluster returns to Ready state", cm.cluster.Name)
	}
	if cm.cluster.Status.Phase == api.ClusterReady {
		var kc kubernetes.Interface
		kc, err = cm.GetAdminClient()
		if err != nil {
			return nil, err
		}
		if upgrade, err := NewKubeVersionGetter(kc, cm.cluster).IsUpgradeRequested(); err != nil {
			return nil, err
		} else if upgrade {
			cm.cluster.Status.Phase = api.ClusterUpgrading
			Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
			return cm.applyUpgrade(dryRun)
		}
	}

	if cm.cluster.Status.Phase == api.ClusterPending {
		a, err := cm.applyCreate(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		nodeGroups, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
		log.Infoln(err)
		for _, ng := range nodeGroups {
			ng.Spec.Replicas = Int32P(int32(0))
			_, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Update(ng)
			if err != nil {
				return nil, err
			}
		}
	}
	{
		a, err := cm.applyScale(dryRun)
		if err != nil && cm.cluster.DeletionTimestamp == nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		a, err := cm.applyDelete(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}
	return acts, nil
}

// Creates network, and creates ready master(s)
func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	lb, err := cm.conn.lbByName(cm.ctx, cm.namer.LoadBalancerName())
	if err == errLBNotFound {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Balancer",
			Message:  fmt.Sprintf("Load Balancer will be created"),
		})
		lb, err = cm.conn.createLoadBalancer(cm.namer.LoadBalancerName())
		if err != nil {
			return
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Balancer",
			Message:  fmt.Sprintf("Load Balancer %q found", lb.Label),
		})
	}

	cm.cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   *lb.IPv4,
		Port: api.DefaultKubernetesBindPort,
	}
	cm.conn.cluster = cm.cluster

	var machines []*clusterv1.Machine
	machines, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	masterMachine, err := api.GetLeaderMachine(machines)
	if err != nil {
		return
	}
	acts = append(acts, api.Action{
		Action:   api.ActionAdd,
		Resource: "Master startup script",
		Message:  "Startup script will be created/updated for master instance",
	})
	if !dryRun {
		if _, err = cm.conn.createOrUpdateStackScript(cm.cluster, masterMachine, "", cm.owner); err != nil {
			return
		}
	}
	if d, _ := cm.conn.instanceIfExists(masterMachine); d == nil {
		Logger(cm.ctx).Info("Creating master instance")
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Master instance %s will be created", masterMachine.Name),
		})
		if !dryRun {
			var masterServer *api.NodeInfo
			nodeAddresses := make([]core.NodeAddress, 0)
			if cm.cluster.Status.Cloud.LoadBalancer.DNS != "" {
				nodeAddresses = append(nodeAddresses, core.NodeAddress{
					Type:    core.NodeExternalDNS,
					Address: cm.cluster.Status.Cloud.LoadBalancer.DNS,
				})
			} else if cm.cluster.Status.Cloud.LoadBalancer.IP != "" {
				nodeAddresses = append(nodeAddresses, core.NodeAddress{
					Type:    core.NodeExternalIP,
					Address: cm.cluster.Status.Cloud.LoadBalancer.IP,
				})
			}

			masterServer, err = cm.conn.CreateInstance(masterMachine.Name, "", masterMachine, cm.owner)
			if err != nil {
				return
			}

			if err = cm.conn.addNodeToBalancer(*lb.Label, masterMachine.Name, masterServer.PrivateIP); err != nil {
				return
			}
			/*if masterServer.PrivateIP != "" {
				nodeAddresses = append(nodeAddresses, core.NodeAddress{
					Type:    core.NodeInternalIP,
					Address: masterServer.PrivateIP,
				})
			}
			if masterServer.PublicIP != "" {
				nodeAddresses = append(nodeAddresses, core.NodeAddress{
					Type:    core.NodeExternalIP,
					Address: masterServer.PublicIP,
				})
			}*/

			if err = cm.cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
				return
			}
		} else {
			acts = append(acts, api.Action{
				Action:   api.ActionNOP,
				Resource: "MasterInstance",
				Message:  "master instance(s) already exist",
			})
		}
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
			return
		}

		var kc kubernetes.Interface
		kc, err = cm.GetAdminClient()
		if err != nil {
			return
		}
		// wait for nodes to start
		if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
			return
		}
		// needed to get master_internal_ip
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
		// need to run ccm
		if err = CreateCredentialSecret(cm.ctx, kc, cm.cluster, cm.owner); err != nil {
			return
		}
		ca, err := NewClusterApi(cm.ctx, cm.cluster, cm.owner, "cloud-provider-system", kc, cm.conn)
		if err != nil {
			return acts, err
		}
		if err := ca.Apply(ControllerManager); err != nil {
			return acts, err
		}
	}

	log.Infof("Adding other master machines")
	client, err := GetClusterClient(cm.ctx, cm.cluster)
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(machines); i++ {
		if _, err := client.ClusterV1alpha1().Machines(cm.cluster.Spec.ClusterAPI.Namespace).Create(machines[i]); err != nil {
			log.Infof("Error creating maching %q in namespace %q", machines[i].Name, cm.cluster.Spec.ClusterAPI.Namespace)
			return acts, err
		}
	}

	cm.cluster.Status.Phase = api.ClusterReady
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
		return
	}

	return acts, err
}

// Scales up/down regular node groups
func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	Logger(cm.ctx).Infoln("scaling machine set")

	var machineSets []*clusterv1.MachineSet
	var existingMachineSet []*clusterv1.MachineSet
	machineSets, err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return nil, err
	}
	var data []byte
	for _, machineSet := range machineSets {
		if machineSet.DeletionTimestamp != nil {
			machineSet.DeletionTimestamp = nil
			if data, err = json.Marshal(machineSet); err != nil {
				return
			}

			if err = bc.Delete(string(data)); err != nil {
				return
			}
			if cm.cluster, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
				return
			}
		}

		if existingMachineSet, err = bc.GetMachineSets(bc.GetContextNamespace()); err != nil {
			return
		}
		if data, err = json.Marshal(machineSet); err != nil {
			return
		}
		found := false
		for _, ems := range existingMachineSet {
			if ems.Name == machineSet.Name {
				found = true
				if err = bc.Apply(string(data)); err != nil {
					return
				}
				break
			}
		}

		if !found {
			if err = bc.CreateMachineSets([]*clusterv1.MachineSet{machineSet}, bc.GetContextNamespace()); err != nil {
				return
			}
		}
	}

	return
}

//Deletes master(s) and releases other cloud resources
func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	Logger(cm.ctx).Infoln("deleting cluster")

	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}
	var nodeInstances *core.NodeList
	nodeInstances, err = kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.RoleNodeKey: "",
		}).String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		Logger(cm.ctx).Infof("node instance not found. Reason: %v", err)
	} else if err == nil {
		for _, mi := range nodeInstances.Items {

			if err = cm.conn.DeleteStackScript(mi.Name, api.RoleNode); err != nil {
				Logger(cm.ctx).Infof("Reason: %v", err)
			}
			err = kc.CoreV1().Nodes().Delete(mi.Name, nil)
			if err != nil {
				Logger(cm.ctx).Infof("Failed to delete node %s. Reason: %s", mi.Name, err)
			}
		}
	}

	var masterInstances *core.NodeList
	masterInstances, err = kc.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			api.RoleMasterKey: "",
		}).String(),
	})
	if err != nil && !kerr.IsNotFound(err) {
		Logger(cm.ctx).Infof("master instance not found. Reason: %v", err)
	} else if err == nil {
		acts = append(acts, api.Action{
			Action:   api.ActionDelete,
			Resource: "MasterInstance",
			Message:  fmt.Sprintf("Will delete master instance with name %v-master", cm.cluster.Name),
		})
		if !dryRun {
			for _, mi := range masterInstances.Items {
				if err = cm.conn.DeleteStackScript(mi.Name, api.RoleMaster); err != nil {
					Logger(cm.ctx).Infof("Reason: %v", err)
				}

				err = cm.conn.DeleteInstanceByProviderID(mi.Spec.ProviderID)
				if err != nil {
					Logger(cm.ctx).Infof("Failed to delete instance %s. Reason: %s", mi.Spec.ProviderID, err)
				}
			}

		}
	}

	_, err = cm.conn.lbByName(cm.ctx, cm.namer.LoadBalancerName())
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

	cm.cluster.Status.Phase = api.ClusterDeleted
	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	Logger(cm.ctx).Infof("Cluster %v deletion is deleted successfully", cm.cluster.Name)
	return
}

func (cm *ClusterManager) applyUpgrade(dryRun bool) (acts []api.Action, err error) {
	var kc kubernetes.Interface
	if kc, err = cm.GetAdminClient(); err != nil {
		return
	}

	var masterMachine *clusterv1.Machine
	masterName := fmt.Sprintf("%v-master", cm.cluster.Name)
	masterMachine, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Get(masterName)
	if err != nil {
		return nil, err
	}

	masterMachine.Spec.Versions.ControlPlane = cm.cluster.Spec.Config.KubernetesVersion
	masterMachine.Spec.Versions.Kubelet = cm.cluster.Spec.Config.KubernetesVersion

	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return nil, err
	}

	var data []byte
	if data, err = json.Marshal(masterMachine); err != nil {
		return
	}
	if err = bc.Apply(string(data)); err != nil {
		return
	}

	// Wait until master updated
	desiredVersion, _ := semver.NewVersion(cm.cluster.ClusterConfig().KubernetesVersion)
	if err = WaitForReadyMasterVersion(cm.ctx, kc, desiredVersion); err != nil {
		return
	}
	// wait for nodes to start
	if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
		return
	}

	var machineSets []*clusterv1.MachineSet
	machineSets, err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, machineSet := range machineSets {
		machineSet.Spec.Template.Spec.Versions.Kubelet = cm.cluster.Spec.Config.KubernetesVersion
		if data, err = json.Marshal(machineSet); err != nil {
			return
		}
		if err = bc.Apply(string(data)); err != nil {
			return
		}
	}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}
	return
}
