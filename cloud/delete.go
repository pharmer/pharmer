package cloud

import (
	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
	"sigs.k8s.io/cluster-api/pkg/util"
)

// DeleteAllWorkerMachines waits for all nodes to be deleted
func DeleteAllWorkerMachines(cm Interface) error {
	log.Infof("Deleting non-controlplane machines")

	clusterClient, err := GetClusterAPIClient(cm.GetCaCertPair(), cm.GetCluster())
	if err != nil {
		return err
	}

	log.Infof("Deleting machine deployments")
	err = deleteMachineDeployments(clusterClient)
	if err != nil {
		log.Infof("failed to delete machine deployments: %v", err)
	}

	log.Infof("Deleting machine sets")
	err = deleteMachineSets(clusterClient)
	if err != nil {
		log.Infof("failed to delete machinesetes: %v", err)
	}

	log.Infof("Deleting machines")
	err = deleteMachines(clusterClient)
	if err != nil {
		log.Infof("failed to delete machines: %v", err)
	}

	log.Infof("successfully deleted non-controlplane machines")
	return nil
}

// deletes machinedeployments in all namespaces
func deleteMachineDeployments(client clientset.Interface) error {
	list, err := client.ClusterV1alpha1().MachineDeployments(corev1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ms := range list.Items {
		err = client.ClusterV1alpha1().MachineDeployments(ms.Namespace).Delete(ms.Name, nil)
		if err != nil {
			return err
		}
	}

	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (done bool, err error) {
		deployList, err := client.ClusterV1alpha1().MachineDeployments(corev1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			log.Infof("failed to list machine deployments: %v", err)
			return false, nil
		}
		if len(deployList.Items) == 0 {
			log.Infof("successfully deleted machine deployments")
			return true, nil
		}
		log.Infof("machine deployments are not deleted yet")
		return false, nil
	})
}

// deletes machinesets in all namespaces
func deleteMachineSets(client clientset.Interface) error {
	list, err := client.ClusterV1alpha1().MachineSets(corev1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ms := range list.Items {
		err = client.ClusterV1alpha1().MachineSets(ms.Namespace).Delete(ms.Name, nil)
		if err != nil {
			return err
		}
	}

	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (done bool, err error) {
		machineSetList, err := client.ClusterV1alpha1().MachineSets(corev1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			log.Infof("failed to list machine sets: %v", err)
			return false, nil
		}
		if len(machineSetList.Items) == 0 {
			log.Infof("successfully deleted machinesets")
			return true, nil
		}
		log.Infof("machinesets are not deleted yet")
		return false, nil
	})
}

// deletes machines in all namespaces
func deleteMachines(client clientset.Interface) error {
	list, err := client.ClusterV1alpha1().Machines(corev1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ms := range list.Items {
		err = client.ClusterV1alpha1().Machines(ms.Namespace).Delete(ms.Name, nil)
		if err != nil {
			return err
		}
	}

	// wait for machines to be deleted
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (done bool, err error) {
		machineList, err := client.ClusterV1alpha1().Machines(corev1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}

		for _, machine := range machineList.Items {
			if !util.IsControlPlaneMachine(&machine) {
				log.Infof("machine %s in namespace %s is not deleted yet", machine.Name, machine.Namespace)
			}
		}

		log.Infof("successfully deleted non-controlplane machines")
		return true, nil
	})
}
