package gce

import (
	"encoding/json"
	"fmt"

	semver "github.com/appscode/go-version"
	. "github.com/appscode/go/context"
	"github.com/appscode/go/log"
	"github.com/appscode/go/wait"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapiGCE "github.com/pharmer/pharmer/apis/v1beta1/gce"
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
		return nil, errors.Errorf("cluster `%s` is in unknown phase", in.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}

	if err = PrepareCloud(cm); err != nil {
		return nil, err
	}

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
			// ignore error if cluster is deleted
			if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
				log.Infoln(err)
			} else {
				return nil, err
			}
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
	if err := cm.SetupCerts(); err != nil {
		return nil, err
	}

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
	masterMachine, err = GetLeaderMachine(cm.ctx, cm.cluster, cm.owner)
	if err != nil {
		return
	}

	loadBalancerIP, err := cm.conn.getLoadBalancer()
	if err != nil {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Load Balancer",
			Message:  "Load Balancer not found",
		})
		if !dryRun {
			if loadBalancerIP, err = cm.conn.createLoadBalancer(masterMachine.Name); err != nil {
				return acts, errors.Wrap(err, "Error creating load balancer")
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Load Balancer",
			Message:  "Load Balancer found",
		})
	}
	cm.cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   loadBalancerIP,
		Port: api.DefaultKubernetesBindPort,
	}

	if loadBalancerIP == "" {
		return nil, errors.Wrap(err, "load balancer can't be empty")
	}

	cm.cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   loadBalancerIP,
		Port: api.DefaultKubernetesBindPort,
	}

	machineSets, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	providerSpec, err := clusterapiGCE.MachineConfigFromProviderSpec(masterMachine.Spec.ProviderSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode machine provider spec")
	}

	if providerSpec.MachineType == "" {
		totalNodes := NodeCount(machineSets)
		sku := "n1-standard-2"

		if totalNodes > 10 {
			sku = "n1-standard-4"
		}
		if totalNodes > 100 {
			sku = "n1-standard-8"
		}
		if totalNodes > 250 {
			sku = "n1-standard-16"
		}
		if totalNodes > 500 {
			sku = "n1-standard-32"
		}

		// update all the master machines
		for _, m := range machines {
			conf, err := clusterapiGCE.MachineConfigFromProviderSpec(m.Spec.ProviderSpec)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to decode provider spec for macine %s", m.Name)
			}
			conf.MachineType = sku

			rawcfg, err := clusterapiGCE.EncodeMachineSpec(conf)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to encode provider spec for machine %s", m.Name)
			}
			m.Spec.ProviderSpec.Value = rawcfg

			_, err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Update(m)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to update machine %q to store", m.Name)
			}
		}
	}

	// needed for master start-up config
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
		cm.cluster.Status.Reason = err.Error()
		err = errors.Wrap(err, ID(cm.ctx))
		return
	}

	Logger(cm.ctx).Info("Preparing Master Instance")

	found, _ = cm.conn.getMasterInstance(masterMachine)

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Master instance with name %v will be created", cm.conn.namer.MasterName()),
		})
		if !dryRun {
			var op1 string
			log.Infoln("Creating Master Instance")
			if op1, err = cm.conn.createMasterIntance(cm.cluster); err != nil {
				return
			}

			if err = cm.conn.waitForZoneOperation(op1); err != nil {
				return
			}

			nodeAddresses := []corev1.NodeAddress{
				{
					Type:    corev1.NodeExternalIP,
					Address: loadBalancerIP,
				},
			}

			Logger(cm.ctx).Info("Waiting for cluster initialization")

			if err = cm.cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
				return acts, errors.Wrap(err, "Error setting controlplane endpoints")
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

	Logger(cm.ctx).Infoln("Creating secret credential...")
	if err = CreateCredentialSecret(cm.ctx, kc, cm.cluster, cm.owner); err != nil {
		return acts, errors.Wrap(err, "Error creating ccm secret credentials")
	}

	ca, err := NewClusterApi(cm.ctx, cm.cluster, cm.owner, "cloud-provider-system", kc, cm.conn)
	if err != nil {
		return acts, errors.Wrap(err, "Error creating cluster-api components")
	}
	var controllerManager string
	controllerManager, err = cm.conn.getControllerManager()
	if err != nil {
		return acts, errors.Wrap(err, "Error creating controller-manager")
	}

	if err := ca.Apply(controllerManager); err != nil {
		return acts, err
	}

	log.Infof("Adding other master machines")
	client, err := GetClusterClient(cm.ctx, cm.cluster)
	if err != nil {
		return nil, err
	}

	for _, m := range machines {
		if m.Name == masterMachine.Name {
			continue
		}
		if _, err := client.ClusterV1alpha1().Machines(cm.cluster.Spec.ClusterAPI.Namespace).Create(m); err != nil {
			log.Infof("Error creating maching %q in namespace %q", m.Name, cm.cluster.Spec.ClusterAPI.Namespace)
			return acts, err
		}
	}

	cm.cluster.Status.Phase = api.ClusterReady
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
		return
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

	// wait for machines to be deleted
	return acts, wait.PollImmediate(RetryInterval, RetryTimeout, func() (done bool, err error) {
		machineList, err := bc.GetMachines(corev1.NamespaceAll)
		for _, machine := range machineList {
			if machine.DeletionTimestamp != nil {
				log.Infof("machine %s is not deleted yet", machine.Name)
				return false, nil
			}
		}
		return true, nil
	})
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

	var nodeDiskNames = make([]string, 0)
	var masterMachines []*clusterv1.Machine

	for _, machine := range machines {
		if !util.IsControlPlaneMachine(machine) {
			acts = append(acts, api.Action{
				Action:   api.ActionDelete,
				Resource: "Machine",
				Message:  fmt.Sprintf("Machine `%v` will be deleted", machine.Spec.Name),
			})
			if !dryRun {
				if err = cm.conn.deleteInstance(machine.Spec.Name); err != nil {
					Logger(cm.ctx).Infof("Error deleting instance `%v`. Reason: %v", machine.Spec.Name, err)
				}

				err = Store(cm.ctx).Owner(cm.owner).Machine(cm.cluster.Name).Delete(machine.Spec.Name)
				if err != nil {
					return nil, err
				}
				err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Delete(machine.Name)
				if err != nil {
					return nil, err
				}
			}
		} else {
			masterMachines = append(masterMachines, machine)
		}
		nodeDiskNames = append(nodeDiskNames, cm.namer.MachineDiskName(machine))
	}

	acts = append(acts, api.Action{
		Action:   api.ActionNOP,
		Resource: "Master Instance",
		Message:  "Deleting master instances",
	})
	if !dryRun {
		Logger(cm.ctx).Infoln("Deleting Master machine")
		if err = cm.conn.deleteMaster(masterMachines); err != nil {
			Logger(cm.ctx).Infof("Error on deleting master. Reason: %v", err)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Cluster Firewall Rule",
		Message:  fmt.Sprintf("%v cluster firewall rule will be deleted", cm.cluster.Name),
	})
	if !dryRun {
		log.Infoln("Deleting Firewall rules")
		if err = cm.conn.deleteFirewalls(); err != nil {
			cm.cluster.Status.Reason = err.Error()
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Master persistent disk",
		Message:  fmt.Sprintf("Will delete master persistent disks"),
	})

	if !dryRun {
		Logger(cm.ctx).Infoln("Deleting disks...")
		if err = cm.conn.deleteDisk(nodeDiskNames); err != nil {
			Logger(cm.ctx).Infof("Error on deleting disk. Reason: %v", err)
		}
	}

	if !dryRun {
		if err := cm.conn.deleteLoadBalancer(); err != nil {
			log.Infof("Error deleting load balancer: %v", err)
		}
	}

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
