package cloud

import (
	"time"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/cloud/utils/kube"
	"pharmer.dev/pharmer/store"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
	"sigs.k8s.io/cluster-api/pkg/util"
)

func Delete(clusterStore store.ClusterStore, name string) (*api.Cluster, error) {
	if name == "" {
		return nil, errors.New("missing Cluster name")
	}

	cluster, err := clusterStore.Get(name)
	if err != nil {
		return nil, errors.Errorf("Cluster `%s` does not exist. Reason: %v", name, err)
	}
	cluster.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	cluster.Status.Phase = api.ClusterDeleting

	return clusterStore.Update(cluster)
}

func ApplyDelete(s *Scope) error {
	log := s.Logger
	log.Info("Deleting cluster")
	cluster := s.Cluster

	err := deleteAllWorkerMachines(s)
	if err != nil {
		log.Error(err, "failed to delete nodes")
	}

	if cluster.Status.Phase == api.ClusterReady {
		cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = s.StoreProvider.Clusters().UpdateStatus(cluster)
	if err != nil {
		return err
	}

	return s.CloudManager.ApplyDelete()
}

// deleteAllWorkerMachines waits for all nodes to be deleted
func deleteAllWorkerMachines(s *Scope) error {
	log := s.Logger
	log.Info("Deleting non-controlplane machines")

	clusterClient, err := kube.GetClusterAPIClient(s.GetCaCertPair(), s.Cluster)
	if err != nil {
		log.Error(err, "failed to get clusterAPI client")
		return err
	}

	log.Info("Deleting machine deployments")
	err = deleteMachineDeployments(clusterClient)
	if err != nil {
		log.Error(err, "failed to delete machine deployments")
	}

	log.Info("Deleting machine sets")
	err = deleteMachineSets(clusterClient)
	if err != nil {
		log.Error(err, "failed to delete machinesetes")
	}

	log.Info("Deleting machines")
	err = deleteMachines(clusterClient)
	if err != nil {
		log.Error(err, "failed to delete machines")
	}

	log.Info("successfully deleted non-controlplane machines")
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
