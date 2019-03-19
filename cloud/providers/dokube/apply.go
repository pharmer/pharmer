package dokube

import (
	"fmt"
	"log"

	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster, cm.owner); err != nil {
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

	acts = append(acts, api.Action{
		Action:   api.ActionAdd,
		Resource: "Kubernetes cluster",
		Message:  fmt.Sprintf("Kubernetes cluster with name %v will be created", cm.cluster.Name),
	})

	if !dryRun {
		cluster, err := cm.conn.createCluster(cm.cluster, cm.owner)

		if err != nil {
			return nil, err
		}

		cm.cluster.Spec.Config.Cloud.Dokube.ClusterID = cluster.ID
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster); err != nil {
			return nil, err
		}

		if err = cm.retrieveClusterStatus(cluster); err != nil {
			return nil, err
		}
		err = cm.StoreCertificate(cm.ctx, cm.conn.client, cm.owner)
		if err != nil {
			log.Println(err)
			return acts, err
		}
		if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
			log.Println(err)
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
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
			return acts, err
		}
	}

	return acts, nil
}

func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	var nodeGroups []*clusterapi.MachineSet
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, ng := range nodeGroups {
		igm := NewDokubeNodeGroupManager(cm.ctx, cm.conn, ng, cm.owner)
		var a2 []api.Action
		a2, err = igm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a2...)
	}
	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return nil, err
	}
	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster)
	if err != nil {
		return nil, err
	}
	return
}

func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}
	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Kubernetes cluster",
		Message:  fmt.Sprintf("%v cluster will be deleted", cm.cluster.Name),
	})
	_, err = cm.conn.client.Kubernetes.Delete(cm.ctx, cm.conn.cluster.Spec.Config.Cloud.Dokube.ClusterID)
	if err != nil {
		return acts, err
	}
	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterDeleted
		_, err = Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster)
		if err != nil {
			return nil, err
		}
	}

	return
}
