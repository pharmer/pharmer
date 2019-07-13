package gce

import (
	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cloud"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func (cm *ClusterManager) PrepareCloud() error {
	// TODO
	if err := cm.conn.importPublicKey(); err != nil {
		return err
	}

	err := ensureNetwork(cm.conn)
	if err != nil {
		return errors.Wrap(err, "failed to ensure network")
	}

	err = ensureFirewallRules(cm.conn)
	if err != nil {
		return errors.Wrap(err, "failed to ensure firewall rules")
	}

	err = ensureLoadBalancer(cm.conn)
	if err != nil {
		return errors.Wrap(err, "failed to ensure load balancer")
	}

	return nil
}

func ensureNetwork(conn *cloudConnector) error {
	found, _ := conn.getNetworks()

	if !found {
		if err := conn.ensureNetworks(); err != nil {
			return err
		}
	}
	return nil
}

func ensureFirewallRules(conn *cloudConnector) error {
	found, _ := conn.getFirewallRules()
	if !found {
		if err := conn.ensureFirewallRules(); err != nil {
			return err
		}
	}
	return nil
}

func ensureLoadBalancer(conn *cloudConnector) error {
	loadBalancerIP, err := conn.getLoadBalancer()
	if err != nil {
		loadBalancerIP, err = conn.createLoadBalancer(conn.Cluster.MasterMachineName(0))
		if err != nil {
			return errors.Wrap(err, "Error creating load balancer")
		}
	}
	conn.Cluster.Status.Cloud.LoadBalancer = api.LoadBalancer{
		IP:   loadBalancerIP,
		Port: api.DefaultKubernetesBindPort,
	}

	if loadBalancerIP == "" {
		return errors.Wrap(err, "load balancer can't be empty")
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

	if err = conn.Cluster.SetClusterAPIEndpoints(nodeAddresses); err != nil {
		return errors.Wrap(err, "Error setting controlplane endpoints")
	}

	return err
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

func (cm *ClusterManager) EnsureMaster(leaderMachine *clusterv1.Machine) error {
	cm.Logger.Info("creating Master instance")

	found, _ := cm.conn.getMasterInstance(leaderMachine)
	if found {
		return nil
	}
	var op1 string

	script, err := cloud.RenderStartupScript(cm, leaderMachine, "", customTemplate)
	if err != nil {
		return err
	}

	if op1, err = cm.conn.createMasterIntance(leaderMachine, script); err != nil {
		return err
	}

	if err = cm.conn.waitForZoneOperation(op1); err != nil {
		return err
	}

	cm.Logger.Info("successfully create master instance")

	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	cm.Logger.Info("deleting cluster")

	machines, err := cm.StoreProvider.Machine(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var nodeDiskNames = make([]string, 0)
	var masterMachines []*clusterv1.Machine

	for _, machine := range machines {
		if !util.IsControlPlaneMachine(machine) {
			if err = cm.conn.deleteInstance(machine.Spec.Name); err != nil {
				log.Infof("Error deleting instance `%v`. Reason: %v", machine.Spec.Name, err)
			}

			err = cm.StoreProvider.Machine(cm.Cluster.Name).Delete(machine.Spec.Name)
			if err != nil {
				return err
			}
			err = cm.StoreProvider.MachineSet(cm.Cluster.Name).Delete(machine.Name)
			if err != nil {
				return err
			}
		} else {
			masterMachines = append(masterMachines, machine)
		}
		nodeDiskNames = append(nodeDiskNames, cm.namer.MachineDiskName(machine))
	}

	log.Infoln("Deleting Master machine")
	if err = cm.conn.deleteMaster(masterMachines); err != nil {
		log.Infof("Error on deleting master. Reason: %v", err)
	}
	log.Infoln("Deleting Firewall rules")
	cm.conn.deleteFirewalls()
	log.Infoln("Deleting disks...")
	if err = cm.conn.deleteDisk(nodeDiskNames); err != nil {
		log.Infof("Error on deleting disk. Reason: %v", err)
	}
	if err := cm.conn.deleteLoadBalancer(); err != nil {
		log.Infof("Error deleting load balancer: %v", err)
	}
	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		return err
	}
	cm.Logger.Info("cluster deleted successfully")

	return nil
}
