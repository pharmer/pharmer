package eks

import (
	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) PrepareCloud() error {
	var found bool
	var err error

	found = cm.conn.isStackExists(cm.namer.GetStackServiceRole())
	if !found {
		if err = cm.conn.createStackServiceRole(); err != nil {
			return err
		}
	}

	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return nil
	}

	found, _ = cm.conn.getPublicKey()

	if !found {
		if err = cm.conn.importPublicKey(); err != nil {
			return err
		}
	}

	found = cm.conn.isStackExists(cm.namer.GetClusterVPC())
	if !found {
		if err = cm.conn.createClusterVPC(); err != nil {
			return err
		}
	}

	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return nil
	}

	found = cm.conn.isControlPlaneExists(cm.Cluster.Name)
	if !found {
		if err = cm.conn.createControlPlane(); err != nil {
			return err
		}
	}

	_, err = store.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		return nil
	}

	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)

	return err
}

func (cm *ClusterManager) ApplyScale() error {
	log.Infoln("scaling node group...")
	var nodeGroups []*clusterapi.MachineSet
	nodeGroups, err := store.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return err
	}
	for _, ng := range nodeGroups {
		igm := NewEKSNodeGroupManager(cm.conn, ng, kc)

		err = igm.Apply()
		if err != nil {
			return err
		}
	}
	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return nil
	}

	_, err = store.StoreProvider.Clusters().Update(cm.Cluster)

	return err
}

func (cm *ClusterManager) ApplyDelete() error {
	log.Infoln("deleting cluster...")
	if cm.Cluster.Status.Phase == api.ClusterReady {
		cm.Cluster.Status.Phase = api.ClusterDeleting
	}
	var found bool
	_, err := store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}
	found = cm.conn.isControlPlaneExists(cm.Cluster.Name)
	if found {
		if err = cm.conn.deleteControlPlane(); err != nil {
			log.Infof("Error on deleting control plane. Reason: %v", err)
		}
	}

	found = cm.conn.isStackExists(cm.namer.GetStackServiceRole())
	if found {
		if err = cm.conn.deleteStack(cm.namer.GetStackServiceRole()); err != nil {
			log.Infof("Error on deleting stack service role. Reason: %v", err)
		}
	}

	found = cm.conn.isStackExists(cm.namer.GetClusterVPC())
	if found {
		if err = cm.conn.deleteStack(cm.namer.GetClusterVPC()); err != nil {
			log.Infof("Error on deleting cluster vpc. Reason: %v", err)
		}
	}

	found, err = cm.conn.getPublicKey()
	if err != nil {
		log.Infoln(err)
	}
	if found {
		if err = cm.conn.deleteSSHKey(); err != nil {
			log.Infof("Error on deleting SSH Key. Reason: %v", err)
		}
	}

	cm.Cluster.Status.Phase = api.ClusterDeleted
	_, err = store.StoreProvider.Clusters().Update(cm.Cluster)

	return err
}
