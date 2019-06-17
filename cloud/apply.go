package cloud

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	semver "gomodules.xyz/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func Apply(opts *options.ApplyConfig) error {
	if opts.ClusterName == "" {
		return errors.New("missing Cluster name")
	}

	cluster, err := store.StoreProvider.Clusters().Get(opts.ClusterName)
	if err != nil {
		return errors.Wrapf(err, "Cluster `%s` does not exist", opts.ClusterName)
	}

	if cluster.Status.Phase == "" {
		return errors.Errorf("Cluster `%s` is in unknown phase", cluster.Name)
	}
	if cluster.Status.Phase == api.ClusterDeleted {
		return nil
	}

	if cluster.Status.Phase == api.ClusterUpgrading {
		return errors.Errorf("Cluster `%s` is upgrading. Retry after Cluster returns to Ready state", cluster.Name)
	}

	cm, err := GetCloudManager(cluster)
	if err != nil {
		return errors.Wrap(err, "failed to get cloudmanager")
	}

	if err = cm.GetCloudConnector(); err != nil {
		return errors.Wrap(err, "failed to get cloud-connector")
	}

	if cluster.Status.Phase == api.ClusterReady {
		var kc kubernetes.Interface
		kc, err = cm.GetAdminClient()
		if err != nil {
			return errors.Wrap(err, "failed to get admin client")
		}
		if upgrade, err := NewKubeVersionGetter(kc, cluster).IsUpgradeRequested(); err != nil {
			return err
		} else if upgrade {
			cluster.Status.Phase = api.ClusterUpgrading
			_, _ = store.StoreProvider.Clusters().UpdateStatus(cluster)
			return ApplyUpgrade(cm)
		}
	}

	if cluster.Status.Phase == api.ClusterPending {
		err := ApplyCreate(cm)
		if err != nil {
			return err
		}

		err = ApplyScale(cm)
		if err != nil {
			return errors.Wrap(err, "failed to scale cluster")
		}
	}

	if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
		err := ApplyDelete(cm)
		if err != nil {
			return err
		}
	}

	return nil
}

func ApplyCreate(cm Interface) error {
	err := cm.PrepareCloud()
	if err != nil {
		return errors.Wrap(err, "failed to prepare cloud infra")
	}

	if !managedProviders.Has(cm.GetCluster().Spec.Config.Cloud.CloudProvider) {
		err = setMasterSKU(cm)
		if err != nil {
			return errors.Wrap(err, "failed to set master sku")
		}

		err = cm.EnsureMaster()
		if err != nil {
			return errors.Wrap(err, "failed to ensure master machine")
		}
	}

	cluster := cm.GetCluster()

	kubeClient, err := cm.GetAdminClient()
	if err != nil {
		return err
	}

	if err = kube.WaitForReadyMaster(kubeClient); err != nil {
		return errors.Wrap(err, " error occurred while waiting for master")
	}

	// create ccm credential
	err = cm.CreateCredentials(kubeClient)
	if err != nil {
		return errors.Wrap(err, "failed to create ccm-credential")
	}

	if !managedProviders.Has(cluster.Spec.Config.Cloud.CloudProvider) {
		err = applyClusterAPI(cm)
		if err != nil {
			return err
		}
	}

	cluster.Status.Phase = api.ClusterReady
	if _, err = store.StoreProvider.Clusters().UpdateStatus(cluster); err != nil {
		return errors.Wrap(err, "failed to update cluster status")
	}

	return nil
}

func applyClusterAPI(cm Interface) error {
	cluster := cm.GetCluster()
	kubeClient, err := cm.GetAdminClient()
	if err != nil {
		return err
	}
	ca, err := NewClusterApi(cm, "cloud-provider-system", kubeClient)
	if err != nil {
		return errors.Wrap(err, "Error creating Cluster-api components")
	}

	clusterAPIcomponents, err := cm.GetClusterAPIComponents()
	if err != nil {
		return errors.Wrap(err, "Error getting clusterAPI components")
	}

	if err := ca.Apply(clusterAPIcomponents); err != nil {
		return err
	}

	log.Infof("Adding other master machines")
	capiClient, err := GetClusterAPIClient(cm.GetCaCertPair(), cluster)
	if err != nil {
		return err
	}

	machines, err := store.StoreProvider.Machine(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list master machines")
	}

	for _, m := range machines {
		_, err := capiClient.ClusterV1alpha1().Machines(cluster.Spec.ClusterAPI.Namespace).Create(m)
		if err != nil && !api.ErrObjectModified(err) && !api.ErrAlreadyExist(err) {
			return errors.Wrapf(err, "failed to crate maching %q in namespace %q",
				m.Name, cluster.Spec.ClusterAPI.Namespace)
		}
	}

	return nil
}

func setMasterSKU(cm Interface) error {
	clusterName := cm.GetCluster().Name

	machines, err := store.StoreProvider.Machine(clusterName).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list machines")
	}

	machineSets, err := store.StoreProvider.MachineSet(clusterName).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	totalNodes := NodeCount(machineSets)

	sku := cm.GetMasterSKU(totalNodes)

	// TODO: should this be in apply? or in create?
	// update all the master machines
	for _, m := range machines {
		providerspec, err := cm.GetDefaultMachineProviderSpec(sku, api.MasterMachineRole)
		if err != nil {
			return err
		}
		m.Spec.ProviderSpec = providerspec

		_, err = store.StoreProvider.Machine(clusterName).Update(m)
		if err != nil {
			return errors.Wrapf(err, "failed to update machine %q to store", m.Name)
		}
	}

	return nil
}

// TODO: simplify
func ApplyScale(cm Interface) error {
	if managedProviders.Has(cm.GetCluster().Spec.Config.Cloud.CloudProvider) {
		return cm.ApplyScale()
	}

	log.Infoln("Scaling Machine Sets")
	cluster := cm.GetCluster()
	var machineSets []*clusterv1.MachineSet
	var existingMachineSet []*clusterv1.MachineSet
	machineSets, err := store.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm, cluster)
	if err != nil {
		return err
	}

	var data []byte
	for _, machineSet := range machineSets {
		if machineSet.DeletionTimestamp != nil {
			machineSet.DeletionTimestamp = nil
			if data, err = json.Marshal(machineSet); err != nil {
				return err
			}
			if err = bc.Delete(string(data)); err != nil {
				return err
			}
			if err = store.StoreProvider.MachineSet(cluster.Name).Delete(machineSet.Name); err != nil {
				return err
			}
		}

		if existingMachineSet, err = bc.GetMachineSets(bc.GetContextNamespace()); err != nil {
			return err
		}

		if data, err = json.Marshal(machineSet); err != nil {
			return err
		}
		found := false
		for _, ems := range existingMachineSet {
			if ems.Name == machineSet.Name {
				found = true
				if err = bc.Apply(string(data)); err != nil {
					return err
				}
				break
			}
		}

		if !found {
			if err = bc.CreateMachineSets([]*clusterv1.MachineSet{machineSet}, bc.GetContextNamespace()); err != nil {
				return err
			}
		}
	}

	_, err = store.StoreProvider.Clusters().UpdateStatus(cluster)
	if err != nil {
		return err
	}
	_, err = store.StoreProvider.Clusters().Update(cluster)
	if err != nil {
		return err
	}

	return nil
}

func ApplyUpgrade(cm Interface) error {
	kc, err := cm.GetAdminClient()
	if err != nil {
		return err
	}

	cluster := cm.GetCluster()
	var masterMachine *clusterv1.Machine
	masterName := fmt.Sprintf("%v-master", cluster.Name)
	masterMachine, err = store.StoreProvider.Machine(cluster.Name).Get(masterName)
	if err != nil {
		return err
	}

	masterMachine.Spec.Versions.ControlPlane = cluster.Spec.Config.KubernetesVersion
	masterMachine.Spec.Versions.Kubelet = cluster.Spec.Config.KubernetesVersion

	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm, cluster)
	if err != nil {
		return err
	}

	var data []byte
	if data, err = json.Marshal(masterMachine); err != nil {
		return err
	}
	if err = bc.Apply(string(data)); err != nil {
		return err
	}

	// Wait until masterMachine is updated
	desiredVersion, _ := semver.NewVersion(cluster.ClusterConfig().KubernetesVersion)
	if err = kube.WaitForReadyMasterVersion(kc, desiredVersion); err != nil {
		return err
	}

	var machineSets []*clusterv1.MachineSet
	machineSets, err = store.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, machineSet := range machineSets {
		machineSet.Spec.Template.Spec.Versions.Kubelet = cluster.Spec.Config.KubernetesVersion
		if data, err = json.Marshal(machineSet); err != nil {
			return err
		}

		if err = bc.Apply(string(data)); err != nil {
			return err
		}
	}

	cluster.Status.Phase = api.ClusterReady
	if _, err = store.StoreProvider.Clusters().UpdateStatus(cluster); err != nil {
		return err
	}

	return err
}

func ApplyDelete(cm Interface) error {
	log.Infoln("Deleting cluster...")
	cluster := cm.GetCluster()

	err := DeleteAllWorkerMachines(cm)
	if err != nil {
		log.Infof("failed to delete nodes: %v", err)
	}

	if cluster.Status.Phase == api.ClusterReady {
		cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = store.StoreProvider.Clusters().UpdateStatus(cluster)
	if err != nil {
		return err
	}

	return cm.ApplyDelete()
}

func NodeCount(machineSets []*clusterv1.MachineSet) int32 {
	var count int32
	for _, machineSet := range machineSets {
		count += *machineSet.Spec.Replicas
	}
	return count
}
