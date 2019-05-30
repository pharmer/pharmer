package cloud

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/cloud/cmds/options"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	semver "gomodules.xyz/version"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func Apply(opts *options.ApplyConfig) ([]api.Action, error) {
	dryRun := opts.DryRun
	if opts.ClusterName == "" {
		return nil, errors.New("missing cluster name")
	}

	cluster, err := store.StoreProvider.Clusters().Get(opts.ClusterName)
	if err != nil {
		return nil, errors.Wrapf(err, "cluster `%s` does not exist", opts.ClusterName)
	}

	var acts []api.Action

	if cluster.Status.Phase == "" {
		return nil, errors.Errorf("cluster `%s` is in unknown phase", cluster.Name)
	}
	if cluster.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}

	if cluster.Status.Phase == api.ClusterUpgrading {
		return nil, errors.Errorf("cluster `%s` is upgrading. Retry after cluster returns to Ready state", cluster.Name)
	}

	cm, err := GetCloudManager(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cloudmanager")
	}

	if err = cm.GetCloudConnector(); err != nil {
		return nil, errors.Wrap(err, "failed to get cloud-connector")
	}

	if cluster.Status.Phase == api.ClusterReady {
		var kc kubernetes.Interface
		kc, err = CreateAdminClient(cm)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get admin client")
		}
		if upgrade, err := NewKubeVersionGetter(kc, cluster).IsUpgradeRequested(); err != nil {
			return nil, err
		} else if upgrade {
			cluster.Status.Phase = api.ClusterUpgrading
			_, _ = store.StoreProvider.Clusters().UpdateStatus(cluster)
			return ApplyUpgrade(dryRun, cm)
		}
	}

	if cluster.Status.Phase == api.ClusterPending {
		a, err := ApplyCreate(dryRun, cm)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
		machines, err := store.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		var replicas int32 = 0
		for _, machine := range machines {
			machine.Spec.Replicas = &replicas
			_, err := store.StoreProvider.MachineSet(cluster.Name).Update(machine)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		a, err := ApplyScale(cm)
		if err != nil {
			// ignore error if cluster is deleted
			if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
				log.Infoln(err)
			} else {
				return nil, err
			}
		}
		acts = append(acts, a...)
	}

	if cluster.DeletionTimestamp != nil && cluster.Status.Phase != api.ClusterDeleted {
		a, err := cm.ApplyDelete(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	return acts, nil
}

func ApplyCreate(dryRun bool, cm Interface) (acts []api.Action, err error) {
	acts, leaderMachine, machines, err := cm.ApplyCreate(dryRun)
	if err != nil {
		return acts, err
	}

	cluster := cm.GetCluster()

	kc, err := CreateAdminClient(cm)
	if err != nil {
		return acts, err
	}

	if err = WaitForReadyMaster(kc); err != nil {
		cluster.Status.Reason = err.Error()
		err = errors.Wrap(err, " error occurred while waiting for master")
		return acts, err
	}

	leaderMachine, err = store.StoreProvider.Machine(cluster.Name).UpdateStatus(leaderMachine)
	if err != nil {
		return
	}

	log.Infoln("Creating secret credential...")
	if err = CreateCredentialSecret(kc, cluster); err != nil {
		return acts, errors.Wrap(err, "Error creating ccm secret credentials")
	}

	conn := cm.GetConnector()
	var controllerManager string
	controllerManager, err = conn.GetControllerManager()

	if err != nil {
		return acts, errors.Wrap(err, "Error creating controller-manager")
	}

	ca, err := NewClusterApi(cm, cluster, "cloud-provider-system", kc, conn)
	if err != nil {
		return acts, errors.Wrap(err, "Error creating cluster-api components")
	}

	if err := ca.Apply(controllerManager); err != nil {
		return acts, err
	}

	log.Infof("Adding other master machines")
	client, err := GetClusterClient(cm, cluster)
	if err != nil {
		return nil, err
	}

	for _, m := range machines {
		if m.Name == leaderMachine.Name {
			continue
		}
		if _, err := client.ClusterV1alpha1().Machines(cluster.Spec.ClusterAPI.Namespace).Create(m); err != nil && !api.ErrObjectModified(err) {
			log.Infof("Error creating maching %q in namespace %q", m.Name, cluster.Spec.ClusterAPI.Namespace)
			return acts, err
		}
	}

	cluster.Status.Phase = api.ClusterReady
	if _, err = store.StoreProvider.Clusters().UpdateStatus(cluster); err != nil {
		return acts, err
	}

	return acts, nil
}

func ApplyScale(cm Interface) (acts []api.Action, err error) {
	log.Infoln("Scaling Machine Sets")
	cluster := cm.GetCluster()
	var machineSets []*clusterv1.MachineSet
	var existingMachineSet []*clusterv1.MachineSet
	machineSets, err = store.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm, cluster)
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
			if err = store.StoreProvider.MachineSet(cluster.Name).Delete(machineSet.Name); err != nil {
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

	_, err = store.StoreProvider.Clusters().UpdateStatus(cluster)
	if err != nil {
		return nil, err
	}
	_, err = store.StoreProvider.Clusters().Update(cluster)
	if err != nil {
		return nil, err
	}

	return acts, nil
}

func ApplyUpgrade(dryRun bool, cm Interface) (acts []api.Action, err error) {
	kc, err := CreateAdminClient(cm)
	if err != nil {
		return acts, err
	}

	cluster := cm.GetCluster()
	var masterMachine *clusterv1.Machine
	masterName := fmt.Sprintf("%v-master", cluster.Name)
	masterMachine, err = store.StoreProvider.Machine(cluster.Name).Get(masterName)
	if err != nil {
		return
	}

	masterMachine.Spec.Versions.ControlPlane = cluster.Spec.Config.KubernetesVersion
	masterMachine.Spec.Versions.Kubelet = cluster.Spec.Config.KubernetesVersion

	var bc clusterclient.Client
	bc, err = GetBooststrapClient(cm, cluster)
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
	desiredVersion, _ := semver.NewVersion(cluster.ClusterConfig().KubernetesVersion)
	if err = WaitForReadyMasterVersion(kc, desiredVersion); err != nil {
		return
	}

	var machineSets []*clusterv1.MachineSet
	machineSets, err = store.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, machineSet := range machineSets {
		machineSet.Spec.Template.Spec.Versions.Kubelet = cluster.Spec.Config.KubernetesVersion
		if data, err = json.Marshal(machineSet); err != nil {
			return
		}

		if err = bc.Apply(string(data)); err != nil {
			return
		}
	}

	if !dryRun {
		cluster.Status.Phase = api.ClusterReady
		if _, err = store.StoreProvider.Clusters().UpdateStatus(cluster); err != nil {
			return
		}
	}

	return
}
