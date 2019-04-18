package gce

import (
	"encoding/json"
	"fmt"
	"log"

	semver "github.com/appscode/go-version"
	. "github.com/appscode/go/context"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	proconfig "github.com/pharmer/pharmer/apis/v1beta1/gce"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
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
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}

	if cm.conn, err = PrepareCloud(cm.ctx, in.Name, cm.owner); err != nil {
		return nil, err
	}
	cm.conn.namer = cm.namer

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
			_, _ = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
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
		machines, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		var replicas int32 = 0
		for _, machine := range machines {
			machine.Spec.Replicas = &replicas
			_, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Update(machine)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		a, err := cm.applyScale(dryRun)
		if err != nil {
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

func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	var found bool
	if !dryRun {
		if err = cm.conn.importPublicKey(); err != nil {
			return
		}
	}

	// TODO: Should we add *IfMissing suffix to all these functions
	found, _ = cm.conn.getNetworks()

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Default Network",
			Message:  "Not found, will add default network with ipv4 range 10.240.0.0/16",
		})
		if !dryRun {
			if err = cm.conn.ensureNetworks(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Default Network",
			Message:  "Found default network with ipv4 range 10.240.0.0/16",
		})
	}

	found, _ = cm.conn.getFirewallRules()
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Default Firewall rule",
			Message:  "default-allow-internal, default-allow-ssh, https rules will be created",
		})
		if !dryRun {
			if err = cm.conn.ensureFirewallRules(); err != nil {
				return
			}
			cm.cluster = cm.conn.cluster
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Default Firewall rule",
			Message:  "default-allow-internal, default-allow-ssh, https rules found",
		})
	}

	// ------------------------------- ASSETS ----------------------------------

	var machines []*clusterv1.Machine
	machines, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, ID(cm.ctx))
		return
	}
	var masterMachine *clusterv1.Machine
	masterMachine, err = api.GetLeaderMachine(machines)
	if err != nil {
		return
	}

	machineSets, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	providerSpec := proconfig.GetGCEMachineProviderSpec(masterMachine.Spec.ProviderSpec)

	if providerSpec.MachineType == "" {
		totalNodes := NodeCount(machineSets)
		providerSpec.MachineType = "n1-standard-2"

		if totalNodes > 10 {
			providerSpec.MachineType = "n1-standard-4"
		}
		if totalNodes > 100 {
			providerSpec.MachineType = "n1-standard-8"
		}
		if totalNodes > 250 {
			providerSpec.MachineType = "n1-standard-16"
		}
		if totalNodes > 500 {
			providerSpec.MachineType = "n1-standard-32"
		}

		masterMachine.Spec.ProviderSpec.Value.Raw, err = json.Marshal(providerSpec)

		masterMachine, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Update(masterMachine)
		if err != nil {
			return
		}
	}
	masterMachine, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Update(masterMachine)

	// needed for master start-up config
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.Wrap(err, ID(cm.ctx))
		return
	}

	Logger(cm.ctx).Info("Preparing Master Instance")

	found, _ = cm.conn.getMasterInstance()

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Master instance with name %v will be created", cm.conn.namer.MasterName()),
		})
		if !dryRun {
			var op1 string
			log.Println("Creating Master Instance")
			if op1, err = cm.conn.createMasterIntance(cm.cluster, masterMachine); err != nil {
				return
			}

			if err = cm.conn.waitForZoneOperation(op1); err != nil {
				return
			}

			var masterInstance *api.NodeInfo
			nodeAddresses := make([]corev1.NodeAddress, 0)
			masterInstance, err = cm.conn.getInstance(cm.conn.namer.MasterName())
			if err != nil {
				return acts, err
			}

			if masterInstance.PrivateIP != "" {
				nodeAddresses = append(nodeAddresses, corev1.NodeAddress{
					Type:    corev1.NodeInternalIP,
					Address: masterInstance.PrivateIP,
				})
			}

			if masterInstance.PublicIP != "" {
				nodeAddresses = append(nodeAddresses, corev1.NodeAddress{
					Type:    corev1.NodeExternalIP,
					Address: masterInstance.PublicIP,
				})
			}

			Logger(cm.ctx).Info("Waiting for cluster initialization")

			if err = cm.cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
				return
			}

		}

	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  "master instance(s) already exist",
		})
	}

	if cm.cluster, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
		return
	}

	//return
	// wait for nodes to start
	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return nil, err
	}

	if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.Wrap(err, ID(cm.ctx))
		return acts, err
	}

	masterMachine, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).UpdateStatus(masterMachine)
	if err != nil {
		return
	}

	// -------------------------------------------------------------------------------------------------------------
	// needed to get master_internal_ip
	cm.cluster.Status.Phase = api.ClusterReady
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
		return
	}
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
		return
	}

	Logger(cm.ctx).Infoln("Creating secret credential...")
	if err = CreateCredentialSecret(cm.ctx, kc, cm.cluster, cm.owner); err != nil {
		return
	}

	if err = proconfig.SetGCEClusterProviderStatus(cm.cluster.Spec.ClusterAPI); err != nil {
		return
	}

	ca, err := NewClusterApi(cm.ctx, cm.cluster, cm.owner, "cloud-provider-system", kc, cm.conn)
	if err != nil {
		return acts, err
	}
	var controllerManager string
	controllerManager, err = cm.conn.getControllerManager()
	if err != nil {
		return acts, err
	}

	if err := ca.Apply(controllerManager); err != nil {
		return acts, err
	}

	return acts, err
}

// TODO: Apparently Not needed
func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	Logger(cm.ctx).Infoln("Scaling Machine Sets")
	var machineSets []*clusterv1.MachineSet
	var existingMachineSet []*clusterv1.MachineSet
	machineSets, err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
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
				return nil, err
			}
			if err = bc.Delete(string(data)); err != nil {
				return nil, err
			}
			if err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Delete(machineSet.Name); err != nil {
				return nil, err
			}
		}

		if existingMachineSet, err = bc.GetMachineSets(bc.GetContextNamespace()); err != nil {
			return nil, err
		}

		if data, err = json.Marshal(machineSet); err != nil {
			return nil, err
		}
		found := false
		for _, ems := range existingMachineSet {
			if ems.Name == machineSet.Name {
				found = true
				if err = bc.Apply(string(data)); err != nil {
					return nil, err
				}
				break
			}
		}

		if !found {
			if err = bc.CreateMachineSets([]*clusterv1.MachineSet{machineSet}, bc.GetContextNamespace()); err != nil {
				return nil, err
			}
		}
	}

	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return nil, err
	}
	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster)
	if err != nil {
		return nil, err
	}

	return
}

func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	Logger(cm.ctx).Infoln("Deleting cluster...")

	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}

	var machines []*clusterv1.Machine
	machines, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var masterMachine *clusterv1.Machine
	masterMachine, err = api.GetLeaderMachine(machines)
	if err != nil {
		return
	}

	var nodeDiskNames = make([]string, 0)

	Logger(cm.ctx).Infoln("Deleting Machines from machineset...")

	for _, machine := range machines {
		if !util.IsControlPlaneMachine(machine) {
			//nodeDiskNames = append(nodeDiskNames, machine.Name)
			//template := cm.namer.InstanceTemplateName(machine.Spec.Template.Spec.SKU)
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Machine",
				//Message:  fmt.Sprintf("%v node group with template %v will be deleted", machine.Name),
				Message: fmt.Sprintf("Machine `%v` will be deleted", machine.Spec.Name),
			})
			if !dryRun {
				if err = cm.conn.deleteInstance(machine.Spec.Name); err != nil {
					Logger(cm.ctx).Infof("Error deleting instance `%v`. Reason: %v", machine.Spec.Name, err)
					//return nil, err
				}

				//if err = cm.conn.deleteOnlyNodeGroup(machine.Name, template); err != nil {
				//	Logger(cm.ctx).Infof("Error on deleting node group. Reason: %v", err)
				//}
				err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Delete(machine.Spec.Name)
				if err != nil {
					return nil, err
				}
				err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Delete(machine.Name)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}

	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return acts, errors.Wrap(err, "Error listing Nodes in cluster")
	}

	Logger(cm.ctx).Infoln("Deleting user defined machines")
	for _, node := range nodes.Items {
		if node.Name != masterMachine.Name {
			//nodeDiskNames = append(nodeDiskNames, node.Name)
			//err = kc.CoreV1().Nodes().Delete(node.Name, metav1.NewDeleteOptions(0))
			err = cm.conn.deleteInstance(node.Name)
			if err != nil {
				return acts, err
			}
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Master Instance",
		Message:  fmt.Sprintf("Found master instance with name %v", cm.namer.MasterName()),
	})
	if !dryRun {
		Logger(cm.ctx).Infoln("Deleting Master machine")
		if err = cm.conn.deleteMaster(); err != nil {
			Logger(cm.ctx).Infof("Error on deleting master. Reason: %v", err)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Cluster Firewall Rule",
		Message:  fmt.Sprintf("%v cluster firewall rule will be deleted", cm.cluster.Name),
	})
	if !dryRun {
		Logger(cm.ctx).Infoln("Deleting Firewall rules")
		if err = cm.conn.deleteFirewalls(); err != nil {
			cm.cluster.Status.Reason = err.Error()
		}
	}

	//if masterMachine.Spec.Template.Spec.ExternalIPType == api.IPTypeReserved {
	//	if !dryRun {
	//		if err = cm.conn.releaseReservedIP(); err != nil {
	//			Logger(cm.ctx).Infof("Error on releasing reserve ip. Reason: %v", err)
	//		}
	//	}
	//}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Master persistent disk",
		Message:  fmt.Sprintf("Will delete master persistent with name %v", cm.namer.MasterPDName()),
	})

	if !dryRun {
		Logger(cm.ctx).Infoln("Deleting disks...")
		if err = cm.conn.deleteDisk(nodeDiskNames); err != nil {
			Logger(cm.ctx).Infof("Error on deleting disk. Reason: %v", err)
		}
	}

	//acts = append(acts, api.Action{
	//	Action:   api.ActionDelete,
	//	Resource: "Route",
	//	Message:  fmt.Sprintf("Route will be delete"),
	//})
	//if !dryRun {
	//	if err = cm.conn.deleteRoutes(); err != nil {
	//		Logger(cm.ctx).Infof("Error on deleting routes. Reason: %v", err)
	//	}
	//}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterDeleted
		_, err := Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster)
		if err != nil {
			return acts, err
		}
	}
	Logger(cm.ctx).Infoln("Cluster deleted successfully...")

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
		return
	}

	masterMachine.Spec.Versions.ControlPlane = cm.cluster.Spec.Config.KubernetesVersion
	masterMachine.Spec.Versions.Kubelet = cm.cluster.Spec.Config.KubernetesVersion

	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return
	}

	var data []byte
	if data, err = json.Marshal(masterMachine); err != nil {
		return
	}
	if err = bc.Apply(string(data)); err != nil {
		return
	}

	// Wait until masterMachine is updated
	desiredVersion, _ := semver.NewVersion(cm.cluster.ClusterConfig().KubernetesVersion)
	if err = WaitForReadyMasterVersion(cm.ctx, kc, desiredVersion); err != nil {
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
