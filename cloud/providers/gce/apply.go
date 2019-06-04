package gce

import (
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func (cm *ClusterManager) PrepareCloud(dryRun bool) ([]api.Action, error) {
	// TODO
	if err := cm.conn.importPublicKey(dryRun); err != nil {
		return nil, err
	}

	var acts []api.Action

	acts, err := ensureNetwork(cm.conn, acts, dryRun)
	if err != nil {
		return acts, errors.Wrap(err, "failed to ensure network")
	}

	acts, err = ensureFirewallRules(cm.conn, acts, dryRun)
	if err != nil {
		return acts, errors.Wrap(err, "failed to ensure firewall rules")
	}

	acts, err = ensureLoadBalancer(cm.conn, acts, dryRun)
	if err != nil {
		return acts, errors.Wrap(err, "failed to ensure load balancer")
	}

	return acts, nil
}

func ensureNetwork(conn *cloudConnector, acts []api.Action, dryRun bool) ([]api.Action, error) {
	found, _ := conn.getNetworks()

	if !found {
		addActs(acts, api.ActionNOP, network, networkNotFoundMessage)
		if err := conn.ensureNetworks(dryRun); err != nil {
			return acts, err
		}
	} else {
		addActs(acts, api.ActionNOP, network, networkFoundMessage)
	}
	return acts, nil
}

func ensureFirewallRules(conn *cloudConnector, acts []api.Action, dryRun bool) ([]api.Action, error) {
	found, _ := conn.getFirewallRules()
	if !found {
		addActs(acts, api.ActionAdd, firewall, firewallNotFoundMessage)
		if err := conn.ensureFirewallRules(dryRun); err != nil {
			return acts, err
		}
	} else {
		addActs(acts, api.ActionNOP, firewall, firewallFoundMessage)
	}
	return acts, nil
}

func ensureLoadBalancer(conn *cloudConnector, acts []api.Action, dryRun bool) ([]api.Action, error) {
	loadBalancerIP, err := conn.getLoadBalancer()
	if err != nil {
		addActs(acts, api.ActionAdd, loadBalancer, loadBalancerNotFoundMessage)
		loadBalancerIP, err = conn.createLoadBalancer(dryRun, conn.Cluster.MasterMachineName(0))
		if err != nil {
			return acts, errors.Wrap(err, "Error creating load balancer")
		}
	} else {
		addActs(acts, api.ActionNOP, loadBalancer, loadBalancerFoundMessage)
	}
	conn.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   loadBalancerIP,
		Port: api.DefaultKubernetesBindPort,
	}

	if loadBalancerIP == "" {
		return acts, errors.Wrap(err, "load balancer can't be empty")
	}

	conn.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   loadBalancerIP,
		Port: api.DefaultKubernetesBindPort,
	}

	nodeAddresses := []corev1.NodeAddress{
		{
			Type:    corev1.NodeExternalIP,
			Address: conn.Cluster.Status.Cloud.LoadBalancer.IP,
		},
	}

	if err = conn.Cluster.SetClusterApiEndpoints(nodeAddresses); err != nil {
		return acts, errors.Wrap(err, "Error setting controlplane endpoints")
	}

	return acts, err
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
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

	return sku
}

func (cm *ClusterManager) EnsureMaster(acts []api.Action, dryRun bool) ([]api.Action, error) {
	leaderMachine, err := GetLeaderMachine(cm.Cluster)
	if err != nil {
		return acts, errors.Wrap(err, "failed to get leader machine")
	}

	found, _ := cm.conn.getMasterInstance(leaderMachine)
	if found {
		return addActs(acts, api.ActionNOP, masterInstance, masterInstanceFoundMessage), nil
	}
	if !dryRun {
		var op1 string
		log.Infoln("Creating Master Instance")

		script, err := RenderStartupScript(cm, leaderMachine, "", customTemplate)
		if err != nil {
			return acts, err
		}

		if op1, err = cm.conn.createMasterIntance(cm.Cluster, script); err != nil {
			return acts, err
		}

		if err = cm.conn.waitForZoneOperation(op1); err != nil {
			return acts, err
		}

	}
	return addActs(acts, api.ActionAdd, masterInstanceMessage, masterInstanceNotFoundMessage), nil
}

func (cm *ClusterManager) ApplyDelete(dryRun bool) (acts []api.Action, err error) {
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
