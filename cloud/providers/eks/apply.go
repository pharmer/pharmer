package eks

import (
	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) PrepareCloud() error {
	var found bool
	var err error

	if cm.Cluster.Spec.ClusterAPI.Spec.ProviderSpec, err = cm.GetDefaultMachineProviderSpec(cm.GetMasterSKU(0), ""); err != nil {
		return err
	}

	if cm.Cluster.Spec.Config.Cloud.InstanceImage, err = cm.conn.DetectInstanceImage(); err != nil {
		return err
	}

	if err = cm.conn.ensureStackServiceRole(); err != nil {
		return err
	}

	found, _ = cm.conn.getPublicKey()

	if !found {
		if err = cm.conn.importPublicKey(); err != nil {
			return err
		}
	}

	if err = cm.conn.ensureClusterVPC(); err != nil {
		return err
	}

	found = cm.conn.isControlPlaneExists(cm.Cluster.Name)
	if !found {
		if err = cm.conn.createControlPlane(); err != nil {
			return err
		}
	}

	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)

	return err
}

func (cm *ClusterManager) ApplyScale() error {
	log.Infoln("scaling node group...")
	var nodeGroups []*clusterapi.MachineSet
	nodeGroups, err := cm.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var kc kubernetes.Interface

	if cm.Cluster.Status.Phase != api.ClusterDeleted {
		kc, err = cm.GetAdminClient()
		if err != nil {
			return err
		}
	}

	for _, ng := range nodeGroups {
		igm := NewEKSNodeGroupManager(cm.Scope, cm.conn, ng, kc)

		err = igm.Apply()
		if err != nil {
			return err
		}
	}
	_, err = cm.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return nil
	}

	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)

	return err
}

func (cm *ClusterManager) ApplyDelete() error {
	found := cm.conn.isControlPlaneExists(cm.Cluster.Name)
	if found {
		if err := cm.conn.deleteControlPlane(); err != nil {
			log.Infof("Error on deleting control plane. Reason: %v", err)
		}
	}

	found = cm.conn.isStackExists(cm.namer.GetStackServiceRole())
	if found {
		if err := cm.conn.deleteStack(cm.namer.GetStackServiceRole()); err != nil {
			log.Infof("Error on deleting stack service role. Reason: %v", err)
		}
	}

	found = cm.conn.isStackExists(cm.namer.GetClusterVPC())
	if found {
		if err := cm.conn.deleteStack(cm.namer.GetClusterVPC()); err != nil {
			log.Infof("Error on deleting cluster vpc. Reason: %v", err)
		}
	}

	found, err := cm.conn.getPublicKey()
	if err != nil {
		log.Infoln(err)
	}
	if found {
		if err := cm.conn.deleteSSHKey(); err != nil {
			log.Infof("Error on deleting SSH Key. Reason: %v", err)
		}
	}

	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = cm.StoreProvider.Clusters().Update(cm.Cluster)

	return err
}
