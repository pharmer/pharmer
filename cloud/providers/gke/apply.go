package gke

import (
	"github.com/pharmer/pharmer/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) PrepareCloud() error {
	err := cm.GetCloudConnector()
	if err != nil {
		return err
	}

	found, _ := cm.conn.getNetworks()
	if !found {
		if err := cm.conn.ensureNetworks(); err != nil {
			return err
		}
	}

	config := cm.Cluster.Spec.Config
	cluster, _ := cm.conn.containerService.Projects.Zones.Clusters.Get(config.Cloud.Project, config.Cloud.Zone, cm.Cluster.Name).Do()
	if cluster == nil {
		cluster, err = encodeCluster(cm.Cluster)
		if err != nil {
			return err
		}

		var op string
		if op, err = cm.conn.createCluster(cluster); err != nil {
			return err
		}
		if err = cm.conn.waitForZoneOperation(op); err != nil {
			return err
		}

		cluster, err = cm.conn.containerService.Projects.Zones.Clusters.Get(config.Cloud.Project, config.Cloud.Zone, cm.Cluster.Name).Do()
		if err != nil {
			return err
		}
		err = cm.StoreCertificate(store.StoreProvider.Certificates(cm.Cluster.Name), cluster)
		if err != nil {
			return err
		}
	}
	if err = cm.retrieveClusterStatus(cluster); err != nil {
		return err
	}

	return nil
}

func (cm *ClusterManager) ApplyScale() error {
	nodeGroups, err := store.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ng := range nodeGroups {
		igm := NewGKENodeGroupManager(cm.conn, ng)
		err = igm.Apply()
		if err != nil {
			return err
		}
	}
	_, err = store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}
	_, err = store.StoreProvider.Clusters().Update(cm.Cluster)
	if err != nil {
		return err
	}
	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	_, err := store.StoreProvider.Clusters().UpdateStatus(cm.Cluster)
	if err != nil {
		return err
	}
	var op string
	op, err = cm.conn.deleteCluster()
	if err = cm.conn.waitForZoneOperation(op); err != nil {
		cm.Cluster.Status.Reason = err.Error()
		return err
	}

	return nil
}
