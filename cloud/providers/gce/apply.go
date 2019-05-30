package gce

import (
	"fmt"

	"github.com/pharmer/pharmer/store"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	clusterapiGCE "github.com/pharmer/pharmer/apis/v1beta1/gce"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func (cm *ClusterManager) ApplyCreate(dryRun bool) (acts []api.Action, leaderMachine *clusterv1.Machine, machines []*clusterv1.Machine, err error) {
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
			cm.Cluster = cm.conn.cluster
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Default Firewall rule",
			Message:  "default-allow-internal, default-allow-ssh, https rules found",
		})
	}

	machines, err = store.StoreProvider.Machine(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, "")
		return
	}

	leaderMachine, err = GetLeaderMachine(cm.Cluster)
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
			if loadBalancerIP, err = cm.conn.createLoadBalancer(leaderMachine.Name); err != nil {
				return acts, nil, nil, errors.Wrap(err, "Error creating load balancer")
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Load Balancer",
			Message:  "Load Balancer found",
		})
	}
	cm.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   loadBalancerIP,
		Port: api.DefaultKubernetesBindPort,
	}

	if loadBalancerIP == "" {
		return nil, nil, nil, errors.Wrap(err, "load balancer can't be empty")
	}

	cm.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   loadBalancerIP,
		Port: api.DefaultKubernetesBindPort,
	}

	machineSets, err := store.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, err
	}

	providerSpec, err := clusterapiGCE.MachineConfigFromProviderSpec(leaderMachine.Spec.ProviderSpec)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to decode machine provider spec")
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
				return nil, nil, nil, errors.Wrapf(err, "failed to decode provider spec for macine %s", m.Name)
			}
			conf.MachineType = sku

			rawcfg, err := clusterapiGCE.EncodeMachineSpec(conf)
			if err != nil {
				return nil, nil, nil, errors.Wrapf(err, "failed to encode provider spec for machine %s", m.Name)
			}
			m.Spec.ProviderSpec.Value = rawcfg

			_, err = store.StoreProvider.Machine(cm.Cluster.Name).Update(m)
			if err != nil {
				return nil, nil, nil, errors.Wrapf(err, "failed to update machine %q to store", m.Name)
			}
		}
	}

	// needed for master start-up config
	if _, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster); err != nil {
		cm.Cluster.Status.Reason = err.Error()
		err = errors.Wrap(err, "")
		return
	}

	log.Info("Preparing Master Instance")

	found, _ = cm.conn.getMasterInstance(leaderMachine)

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Master Instance",
			Message:  fmt.Sprintf("Master instance with name %v will be created", leaderMachine.Name),
		})
		if !dryRun {
			var op1 string
			log.Infoln("Creating Master Instance")
			if op1, err = cm.conn.createMasterIntance(cm.Cluster); err != nil {
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

			log.Info("Waiting for cluster initialization")

			if err = cm.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
				return acts, nil, nil, errors.Wrap(err, "Error setting controlplane endpoints")
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Master Instance",
			Message:  "master instance(s) already exist",
		})
	}

	return acts, leaderMachine, machines, err
}

//func ensureNetwork() {
//	found, _ = cm.conn.getNetworks()
//
//	if !found {
//		addActs(acts, api.ActionNOP, network, foundNetworkMessage)
//		if !dryRun {
//			if err = cm.conn.ensureNetworks(); err != nil {
//				return
//			}
//		}
//	} else {
//		addActs(acts, api.ActionNOP, network, notFoundNetworkMessage)
//	}
//}

// TODO: Apparently Not needed

func (cm *ClusterManager) ApplyDelete(dryRun bool) (acts []api.Action, err error) {
	log.Infoln("Deleting cluster...")

	err = DeleteAllWorkerMachines(cm, cm.Cluster)
	if err != nil {
		log.Infof("failed to delete nodes: %v", err)
	}

	if cm.Cluster.Status.Phase == api.ClusterReady {
		cm.Cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return
	}

	var machines []*clusterv1.Machine
	machines, err = store.StoreProvider.Machine(cm.Cluster.Name).List(metav1.ListOptions{})
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
					log.Infof("Error deleting instance `%v`. Reason: %v", machine.Spec.Name, err)
				}

				err = store.StoreProvider.Machine(cm.Cluster.Name).Delete(machine.Spec.Name)
				if err != nil {
					return nil, err
				}
				err = store.StoreProvider.MachineSet(cm.Cluster.Name).Delete(machine.Name)
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
		log.Infoln("Deleting Master machine")
		if err = cm.conn.deleteMaster(masterMachines); err != nil {
			log.Infof("Error on deleting master. Reason: %v", err)
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Cluster Firewall Rule",
		Message:  fmt.Sprintf("%v cluster firewall rule will be deleted", cm.Cluster.Name),
	})
	if !dryRun {
		log.Infoln("Deleting Firewall rules")
		if err = cm.conn.deleteFirewalls(); err != nil {
			cm.Cluster.Status.Reason = err.Error()
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Master persistent disk",
		Message:  fmt.Sprintf("Will delete master persistent disks"),
	})

	if !dryRun {
		log.Infoln("Deleting disks...")
		if err = cm.conn.deleteDisk(nodeDiskNames); err != nil {
			log.Infof("Error on deleting disk. Reason: %v", err)
		}
	}

	if !dryRun {
		if err := cm.conn.deleteLoadBalancer(); err != nil {
			log.Infof("Error deleting load balancer: %v", err)
		}
	}

	if !dryRun {
		cm.Cluster.Status.Phase = api.ClusterDeleted
		_, err := store.StoreProvider.Clusters().Update(cm.Cluster)
		if err != nil {
			return acts, err
		}
	}
	log.Infoln("Cluster deleted successfully...")

	return
}
