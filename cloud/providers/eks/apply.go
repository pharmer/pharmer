package eks

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) PrepareCloud() error {
	var found bool
	var err error

	log := cm.Logger

	if cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec, err = cm.GetDefaultMachineProviderSpec(cm.GetMasterSKU(0), ""); err != nil {
		log.Error(err, "failed to get provider spec")
		return err
	}

	if cm.Cluster.Spec.Config.Cloud.InstanceImage, err = cm.conn.DetectInstanceImage(); err != nil {
		log.Error(err, "failed to detect instance image")
		return err
	}

	if err = cm.conn.ensureStackServiceRole(); err != nil {
		log.Error(err, "failed to ensure stack service role")
		return err
	}

	found, _ = cm.conn.getPublicKey()

	if !found {
		if err = cm.conn.importPublicKey(); err != nil {
			log.Error(err, "failed to import public key")
			return err
		}
	}

	if err = cm.conn.ensureClusterVPC(); err != nil {
		log.Error(err, "failed to ensure vpc")
		return err
	}

	found = cm.conn.isControlPlaneExists(cm.Cluster.Name)
	if !found {
		if err = cm.conn.createControlPlane(); err != nil {
			log.Error(err, "failed to create control plane")
			return err
		}
	}

	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster in store")
		return err
	}

	return nil
}

func (cm *ClusterManager) ApplyScale() error {
	log := cm.Logger

	var nodeGroups []*clusterapi.MachineSet
	nodeGroups, err := cm.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list machineset")
		return err
	}

	var kc kubernetes.Interface

	if cm.Cluster.Status.Phase != api.ClusterDeleted {
		kc, err = cm.GetAdminClient()
		if err != nil {
			log.Error(err, "failed to get admin client")
			return err
		}
	}

	for _, ng := range nodeGroups {
		igm := NewEKSNodeGroupManager(cm.Scope, cm.conn, ng, kc)

		err = igm.Apply()
		if err != nil {
			log.Error(err, "failed to apply node group")
			return err
		}
	}

	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster in store")
		return err
	}
	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	log := cm.Logger

	err := cm.ApplyScale()
	log.Error(err, "error scaling cluster. Skipping error.")

	found := cm.conn.isControlPlaneExists(cm.Cluster.Name)
	if found {
		if err := cm.conn.deleteControlPlane(); err != nil {
			log.Error(err, "error on deleting control plane")
		}
	}

	found = cm.conn.isStackExists(cm.namer.GetStackServiceRole())
	if found {
		if err := cm.conn.deleteStack(cm.namer.GetStackServiceRole()); err != nil {
			log.Error(err, "error on deleting stack service role")
		}
	}

	found = cm.conn.isStackExists(cm.namer.GetClusterVPC())
	if found {
		if err := cm.conn.deleteStack(cm.namer.GetClusterVPC()); err != nil {
			log.Error(err, "Error on deleting cluster vpc")
		}
	}

	found, err = cm.conn.getPublicKey()
	if err != nil {
		log.Error(err, "error getting public key")
	}
	if found {
		if err := cm.conn.deleteSSHKey(); err != nil {
			log.Error(err, "error on deleting SSH Key.")
		}
	}

	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		log.Error(err, "failed to update cluster in store")
		return err
	}
	return nil
}
