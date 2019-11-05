/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cloud

import (
	"encoding/json"
	"fmt"
	"time"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud/utils/kube"

	"github.com/pkg/errors"
	semver "gomodules.xyz/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func Apply(scope *Scope) error {
	if scope.Cluster.DeletionTimestamp == nil {
		scope.Logger = scope.Logger.WithName("[apply-cluster]")
	} else {
		scope.Logger = scope.Logger.WithName("[delete-cluster]")
	}
	log := scope.Logger

	log.Info("applying cluster")

	cluster := scope.Cluster
	if cluster.Status.Phase == "" {
		log.Info("cluster is in unknown phase")
		return errors.Errorf("Cluster `%s` is in unknown phase", cluster.Name)
	}
	if cluster.Status.Phase == api.ClusterDeleted {
		log.Info("cluster is already deleted, ignoring")
		return nil
	}

	if cluster.Status.Phase == api.ClusterUpgrading {
		log.Info("cluster is upgrading")
		return errors.Errorf("Cluster `%s` is upgrading. Retry after Cluster returns to Ready state", cluster.Name)
	}

	_, err := scope.GetCloudManager()
	if err != nil {
		log.Error(err, "failed to get cloud manager")
		return err
	}
	err = scope.CloudManager.SetCloudConnector()
	if err != nil {
		log.Error(err, "failed to set cloud-connector")
		return err
	}

	if cluster.Status.Phase == api.ClusterPending {
		err := ApplyCreate(scope)
		if err != nil {
			log.Error(err, "failed in applycreate")
			return err
		}
		err = ApplyScale(scope)
		if err != nil {
			log.Error(err, "failed to scale cluster")
			return err
		}
	}

	if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
		machineSets, err := scope.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			log.Error(err, "failed to list machinesets from store")
			return err
		}
		var replica int32 = 0
		for _, ng := range machineSets {
			ng.Spec.Replicas = &replica
			ng.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			_, err := scope.StoreProvider.MachineSet(cluster.Name).Update(ng)
			if err != nil {
				log.Error(err, "failed to update machinesets in store")
				return err
			}
		}

		err = ApplyDelete(scope)
		if err != nil {
			log.Error(err, "failed in ApplyDelete")
			return err
		}
	}

	log.Info("successfully applied cluster")

	return nil
}

func ApplyCreate(scope *Scope) error {
	log := scope.Logger
	cm, err := scope.GetCloudManager()
	if err != nil {
		return err
	}

	err = cm.PrepareCloud()
	if err != nil {
		log.Error(err, "failed to prepare cloud infrastructure")
		return err
	}

	if !api.ManagedProviders.Has(cm.GetCluster().Spec.Config.Cloud.CloudProvider) {
		err = setMasterSKU(scope)
		if err != nil {
			log.Error(err, "failed to set default master sku")
			return err
		}

		machine, err := getLeaderMachine(scope.StoreProvider.Machine(scope.Cluster.Name), scope.Cluster.Name)
		if err != nil {
			log.Error(err, "failed to get leader machine")
			return err
		}

		err = cm.EnsureMaster(machine)
		if err != nil {
			log.Error(err, "failed to create master machine")
			return err
		}
	}

	cluster := cm.GetCluster()

	kubeClient, err := cm.GetAdminClient()
	if err != nil {
		log.Error(err, "failed to get admin client")
		return err
	}

	if err = kube.WaitForReadyMaster(log, kubeClient); err != nil {
		log.Error(err, "failed waiting for master to be ready")
		return err
	}

	// create ccm credential
	err = cm.CreateCredentials(kubeClient)
	if err != nil {
		log.Error(err, "failed to create ccm-credential")
		return err
	}

	if !api.ManagedProviders.Has(cluster.Spec.Config.Cloud.CloudProvider) {
		err = applyClusterAPI(scope)
		if err != nil {
			log.Error(err, "failed to create cluster api components")
			return err
		}
	}

	ns, err := scope.AdminClient.CoreV1().Namespaces().Get(metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "failed to get `kube-system` namespace")
		return err
	}
	if cluster, err = scope.StoreProvider.Clusters().UpdateUUID(cluster, string(ns.UID)); err != nil {
		log.Error(err, "failed to update cluster uuid in store")
		return err
	}

	cluster.Status.Phase = api.ClusterReady
	if _, err = scope.StoreProvider.Clusters().UpdateStatus(cluster); err != nil {
		log.Error(err, "failed to update cluster status in store")
		return err
	}

	log.Info("successful")

	return nil
}

func applyClusterAPI(s *Scope) error {
	cluster := s.Cluster
	ca, err := NewClusterAPI(s, "cloud-provider-system")
	if err != nil {
		s.Logger.Error(err, "failed to create cluster-api clients")
		return err
	}
	ca.Logger = s.Logger.WithName("[cluster-api]")
	log := ca.Logger

	clusterAPIcomponents, err := s.CloudManager.GetClusterAPIComponents()
	if err != nil {
		log.Error(err, "failed to get clusterAPI components")
		return err
	}

	if err := ca.Apply(clusterAPIcomponents); err != nil {
		log.Error(err, "failed to apply cluster api components")
		return err
	}

	log.Info("adding other master machines")

	clusterEndpoint := s.Cluster.APIServerURL()
	if clusterEndpoint == "" {
		return errors.Errorf("failed to detect api server url for Cluster %s", s.Cluster.Name)
	}

	capiClient, err := kube.GetClusterAPIClient(s.StoreProvider.Certificates(cluster.Name), clusterEndpoint)
	if err != nil {
		log.Error(err, "failed to get cluster api client")
		return err
	}

	machines, err := s.StoreProvider.Machine(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list master machines")
		return err
	}

	for _, m := range machines {
		_, err := capiClient.ClusterV1alpha1().Machines(cluster.Spec.ClusterAPI.Namespace).Create(m)
		if err != nil && !api.ErrObjectModified(err) && !api.ErrAlreadyExist(err) {
			log.Error(err, "failed ot create machine", "namespace", cluster.Spec.ClusterAPI.Namespace, "machine-name", m.Name)
			return err
		}
	}

	return nil
}

func setMasterSKU(scope *Scope) error {
	log := scope.Logger

	clusterName := scope.Cluster.Name
	cm := scope.CloudManager

	machines, err := scope.StoreProvider.Machine(clusterName).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list machines from store")
		return err
	}

	machineSets, err := scope.StoreProvider.MachineSet(clusterName).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list machinesets from store")
		return err
	}

	totalNodes := nodeCount(machineSets)

	sku := cm.GetMasterSKU(totalNodes)

	// update all the master machines
	for _, m := range machines {
		providerspec, err := cm.GetDefaultMachineProviderSpec(sku, api.MasterMachineRole)
		if err != nil {
			log.Error(err, "failed to get default provider spec")
			return err
		}
		m.Spec.ProviderSpec = providerspec

		_, err = scope.StoreProvider.Machine(clusterName).Update(m)
		if err != nil {
			log.Error(err, "failed to update machine to store", "machine-name", m.Name)
			return err
		}
	}

	return nil
}

func ApplyScale(s *Scope) error {
	log := s.Logger

	log.Info("scaling Machine Sets")

	if api.ManagedProviders.Has(s.Cluster.CloudProvider()) {
		return s.CloudManager.ApplyScale()
	}

	cluster := s.Cluster
	machineSets, err := s.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list machineset")
		return err
	}

	bc, err := kube.GetBooststrapClient(s.Cluster, s.GetCaCertPair())
	if err != nil {
		log.Error(err, "failed to get bootstrap clients")
		return err
	}

	clusterEndpoint := s.Cluster.APIServerURL()
	if clusterEndpoint == "" {
		return errors.Errorf("failed to detect api server url for Cluster %s", s.Cluster.Name)
	}

	clusterClient, err := kube.GetClusterAPIClient(s.StoreProvider.Certificates(s.Cluster.Name), clusterEndpoint)
	if err != nil {
		log.Error(err, "failed to get cluster api client")
		return err
	}

	for _, machineSet := range machineSets {
		if machineSet.DeletionTimestamp != nil {
			err = clusterClient.ClusterV1alpha1().MachineSets(machineSet.Namespace).Delete(machineSet.Name, nil)
			if err != nil {
				log.Error(err, "failed to delete machinesets")
				return err
			}
			if err = s.StoreProvider.MachineSet(cluster.Name).Delete(machineSet.Name); err != nil {
				log.Error(err, "failed to delete machinesets")
				return err
			}
			continue
		}

		data, err := json.Marshal(machineSet)
		if err != nil {
			log.Error(err, "failed to json marshal machineset")
			return err
		}
		if err = bc.Apply(string(data)); err != nil {
			log.Error(err, "failed to apply machineset")
			return err
		}
	}

	log.Info("successfully scaled machineset")

	return nil
}

func ApplyUpgrade(s *Scope) error {
	s.Logger = s.Logger.WithName("[apply-upgrade]")

	kc, err := s.GetAdminClient()
	if err != nil {
		return err
	}

	Cluster := s.Cluster
	var masterMachine *clusterapi.Machine
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

	var machineSets []*clusterapi.MachineSet
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

func nodeCount(machineSets []*clusterapi.MachineSet) int32 {
	var count int32
	for _, machineSet := range machineSets {
		count += *machineSet.Spec.Replicas
	}
	return count
}
