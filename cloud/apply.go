package cloud

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/cloud/utils/kube"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	semver "gomodules.xyz/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func Apply(opts *options.ApplyConfig, storeProvider store.ResourceInterface) error {
	if opts.ClusterName == "" {
		return errors.New("missing Cluster name")
	}

	cluster, err := storeProvider.Clusters().Get(opts.ClusterName)
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

	scope := NewScope(NewScopeParams{Cluster: cluster, StoreProvider: storeProvider})
	_, err = scope.GetCloudManager()
	if err != nil {
		return err
	}
	err = scope.CloudManager.SetCloudConnector()
	if err != nil {
		return err
	}

	if cluster.Status.Phase == api.ClusterPending {
		err := ApplyCreate(scope)
		if err != nil {
			return err
		}
		err = ApplyScale(scope)
		if err != nil {
			return errors.Wrap(err, "failed to scale Cluster")
		}
	}

	if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
		machineSets, err := scope.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		var replica int32 = 0
		for _, ng := range machineSets {
			ng.Spec.Replicas = &replica
			ng.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			_, err := scope.StoreProvider.MachineSet(cluster.Name).Update(ng)
			if err != nil {
				return err
			}
		}

		if api.ManagedProviders.Has(cluster.CloudProvider()) {
			err = ApplyScale(scope)
			if err != nil {
				return errors.Wrap(err, "failed to scale Cluster")
			}
		}

		err = ApplyDelete(scope)
		if err != nil {
			return err
		}
	}

	return nil
}

func ApplyCreate(scope *Scope) error {
	cm, err := scope.GetCloudManager()
	if err != nil {
		return err
	}

	err = cm.PrepareCloud()
	if err != nil {
		return errors.Wrap(err, "failed to prepare cloud infra")
	}

	if !api.ManagedProviders.Has(cm.GetCluster().Spec.Config.Cloud.CloudProvider) {
		err = setMasterSKU(scope)
		if err != nil {
			return errors.Wrap(err, "failed to set master sku")
		}

		machine, err := getLeaderMachine(scope.StoreProvider.Machine(scope.Cluster.Name), scope.Cluster.Name)
		if err != nil {
			return err
		}

		err = cm.EnsureMaster(machine)
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

	if !api.ManagedProviders.Has(cluster.Spec.Config.Cloud.CloudProvider) {
		err = applyClusterAPI(scope)
		if err != nil {
			return err
		}
	}

	cluster.Status.Phase = api.ClusterReady
	if _, err = scope.StoreProvider.Clusters().UpdateStatus(cluster); err != nil {
		return errors.Wrap(err, "failed to update cluster status")
	}

	return nil
}

func applyClusterAPI(s *Scope) error {
	cluster := s.Cluster
	ca, err := NewClusterAPI(s, "cloud-provider-system")
	if err != nil {
		return errors.Wrap(err, "Error creating Cluster-api components")
	}

	clusterAPIcomponents, err := s.CloudManager.GetClusterAPIComponents()
	if err != nil {
		return errors.Wrap(err, "Error getting clusterAPI components")
	}

	if err := ca.Apply(clusterAPIcomponents); err != nil {
		return err
	}

	log.Infof("Adding other master machines")
	capiClient, err := kube.GetClusterAPIClient(s.GetCaCertPair(), cluster)
	if err != nil {
		return err
	}

	machines, err := s.StoreProvider.Machine(cluster.Name).List(metav1.ListOptions{})
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

func setMasterSKU(scope *Scope) error {
	clusterName := scope.Cluster.Name
	cm := scope.CloudManager

	machines, err := scope.StoreProvider.Machine(clusterName).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list machines")
	}

	machineSets, err := scope.StoreProvider.MachineSet(clusterName).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	totalNodes := nodeCount(machineSets)

	sku := cm.GetMasterSKU(totalNodes)

	// update all the master machines
	for _, m := range machines {
		providerspec, err := cm.GetDefaultMachineProviderSpec(sku, api.MasterMachineRole)
		if err != nil {
			return err
		}
		m.Spec.ProviderSpec = providerspec

		_, err = scope.StoreProvider.Machine(clusterName).Update(m)
		if err != nil {
			return errors.Wrapf(err, "failed to update machine %q to store", m.Name)
		}
	}

	return nil
}

func ApplyScale(s *Scope) error {
	log.Infoln("Scaling Machine Sets")

	if api.ManagedProviders.Has(s.Cluster.CloudProvider()) {
		return s.CloudManager.ApplyScale()
	}

	cluster := s.Cluster
	machineSets, err := s.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	bc, err := kube.GetBooststrapClient(s.Cluster, s.GetCaCertPair())
	if err != nil {
		return err
	}

	clusterClient, err := kube.GetClusterAPIClient(s.GetCaCertPair(), s.Cluster)
	if err != nil {
		return err
	}

	for _, machineSet := range machineSets {
		if machineSet.DeletionTimestamp != nil {
			err = clusterClient.ClusterV1alpha1().MachineSets(machineSet.Namespace).Delete(machineSet.Name, nil)
			if err != nil {
				return err
			}
			if err = s.StoreProvider.MachineSet(cluster.Name).Delete(machineSet.Name); err != nil {
				return err
			}
			continue
		}

		data, err := json.Marshal(machineSet)
		if err != nil {
			return err
		}
		if err = bc.Apply(string(data)); err != nil {
			return err
		}
	}

	return nil
}

func ApplyUpgrade(s *Scope) error {
	kc, err := s.GetAdminClient()
	if err != nil {
		return err
	}

	Cluster := s.Cluster
	var masterMachine *clusterv1.Machine
	masterName := fmt.Sprintf("%v-master", Cluster.Name)
	masterMachine, err = s.StoreProvider.Machine(Cluster.Name).Get(masterName)
	if err != nil {
		return err
	}

	masterMachine.Spec.Versions.ControlPlane = Cluster.Spec.Config.KubernetesVersion
	masterMachine.Spec.Versions.Kubelet = Cluster.Spec.Config.KubernetesVersion

	var bc clusterclient.Client
	bc, err = kube.GetBooststrapClient(s.Cluster, s.GetCaCertPair())
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
	desiredVersion, _ := semver.NewVersion(s.Cluster.ClusterConfig().KubernetesVersion)
	if err = kube.WaitForReadyMasterVersion(kc, desiredVersion); err != nil {
		return err
	}

	var machineSets []*clusterv1.MachineSet
	machineSets, err = s.StoreProvider.MachineSet(Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, machineSet := range machineSets {
		machineSet.Spec.Template.Spec.Versions.Kubelet = Cluster.Spec.Config.KubernetesVersion
		if data, err = json.Marshal(machineSet); err != nil {
			return err
		}

		if err = bc.Apply(string(data)); err != nil {
			return err
		}
	}

	Cluster.Status.Phase = api.ClusterReady
	if _, err = s.StoreProvider.Clusters().UpdateStatus(Cluster); err != nil {
		return err
	}

	return err
}

func nodeCount(machineSets []*clusterv1.MachineSet) int32 {
	var count int32
	for _, machineSet := range machineSets {
		count += *machineSet.Spec.Replicas
	}
	return count
}
