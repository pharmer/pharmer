package gke

import (
	"fmt"

	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	container "google.golang.org/api/container/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error
	var acts []api.Action

	if in.Status.Phase == "" {
		return nil, errors.Errorf("cluster `%s` is in unknown phase", cm.cluster.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	/*if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}*/
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster); err != nil {
		return nil, err
	}

	if cm.cluster.Status.Phase == api.ClusterPending {
		a, err := cm.applyCreate(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	{
		a, err := cm.applyScale(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		a, err := cm.applyDelete(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	return acts, nil
}

func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	found, _ := cm.conn.getNetworks()
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Default Network",
			Message:  "Not found, will add default network with ipv4 range 10.240.0.0/16",
		})
		if !dryRun {
			if err = cm.conn.ensureNetworks(); err != nil {
				return acts, err
			}
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionAdd,
		Resource: "Kubernetes cluster",
		Message:  fmt.Sprintf("Kubernetes cluster with name %v will be created", cm.cluster.Name),
	})
	var cluster *container.Cluster

	cluster, _ = cm.conn.containerService.Projects.Zones.Clusters.Get(cm.conn.cluster.Spec.Cloud.Project, cm.conn.cluster.Spec.Cloud.Zone, cm.cluster.Name).Do()
	if cluster == nil && !dryRun {
		if cluster, err = encodeCluster(cm.ctx, cm.cluster, cm.owner); err != nil {
			return acts, err
		}

		var op string
		if op, err = cm.conn.createCluster(cluster); err != nil {
			return acts, err
		}
		if err = cm.conn.waitForZoneOperation(op); err != nil {
			cm.cluster.Status.Reason = err.Error()
			return acts, err
		}

		cluster, err = cm.conn.containerService.Projects.Zones.Clusters.Get(cm.conn.cluster.Spec.Cloud.Project, cm.conn.cluster.Spec.Cloud.Zone, cm.cluster.Name).Do()
		if err != nil {
			return acts, err
		}
		cm.retrieveClusterStatus(cluster)
		err = cm.StoreCertificate(cluster)
		if err != nil {
			return acts, err
		}
		if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
			return acts, err
		}
		var kc kubernetes.Interface
		if kc, err = cm.GetAdminClient(); err != nil {
			return acts, err
		}
		if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
			cm.cluster.Status.Reason = err.Error()
			return acts, err
		}

		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster); err != nil {
			return acts, err
		}
	}

	return acts, nil
}

func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	var nodeGroups []*api.NodeGroup
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).NodeGroups(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, ng := range nodeGroups {
		igm := NewGKENodeGroupManager(cm.ctx, cm.conn, ng)
		var a2 []api.Action
		a2, err = igm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a2...)
	}
	Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	Store(cm.ctx).Clusters().Update(cm.cluster)
	return
}

func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}
	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Kubernetes cluster",
		Message:  fmt.Sprintf("%v cluster will be deleted", cm.cluster.Name),
	})
	var op string
	op, err = cm.conn.deleteCluster()
	if err = cm.conn.waitForZoneOperation(op); err != nil {
		cm.cluster.Status.Reason = err.Error()
		return acts, err
	}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterDeleted
		Store(cm.ctx).Clusters().Update(cm.cluster)
	}

	return
}
