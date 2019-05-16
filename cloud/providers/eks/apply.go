package eks

import (
	"fmt"
	"time"

	"github.com/appscode/go/log"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	preTagDelay = 5 * time.Second
)

func (cm *ClusterManager) Apply(in *api.Cluster, dryRun bool) ([]api.Action, error) {
	var err error
	var acts []api.Action

	if in.Status.Phase == "" {
		return nil, errors.Errorf("cluster `%s` is in unknown phase", in.Name)
	}
	if in.Status.Phase == api.ClusterDeleted {
		return nil, nil
	}
	cm.cluster = in
	cm.namer = namer{cluster: cm.cluster}
	if cm.ctx, err = LoadCACertificates(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	cm.conn.namer = cm.namer

	if cm.cluster.Spec.Config.Cloud.InstanceImage, err = cm.conn.DetectInstanceImage(); err != nil {
		return nil, err
	}

	if cm.cluster.Status.Phase == api.ClusterPending {
		a, err := cm.applyCreate(dryRun)
		if err != nil {
			return nil, err
		}
		acts = append(acts, a...)
	}

	if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
		nodeGroups, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		var replica int32 = 0
		for _, ng := range nodeGroups {
			ng.Spec.Replicas = &replica
			_, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).Update(ng)
			if err != nil {
				return nil, err
			}
		}
	}

	{
		a, err := cm.applyScale(dryRun)
		if err != nil {
			if cm.cluster.DeletionTimestamp != nil && cm.cluster.Status.Phase != api.ClusterDeleted {
				log.Infoln(err)
			} else {
				return nil, err
			}
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

	// detect-master
	// wait-master: via curl call polling
	// build-config

	//  # KUBE_SHARE_MASTER is used to add nodes to an existing master
	//  if [[ "${KUBE_SHARE_MASTER:-}" == "true" ]]; then
	//    detect-master
	//    start-nodes
	//    wait-nodes
	//  else
	//    start-master
	//    start-nodes
	//    wait-nodes
	//    wait-master
	//
	//    # Build ~/.kube/config
	//    build-config
	//  fi
	// check-cluster
	return acts, nil
}

func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	var found bool

	if found, err = cm.conn.isStackExists(cm.namer.GetStackServiceRole()); err != nil {
		return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "IAM Profile",
			Message:  "IAM profile will be created",
		})
		if !dryRun {
			if err = cm.conn.createStackServiceRole(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "IAM Profile",
			Message:  "IAM profile found",
		})
	}

	if found, err = cm.conn.getPublicKey(); err != nil {
		//return
	}

	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "PublicKey",
			Message:  "Public key will be imported",
		})
		if !dryRun {
			if err = cm.conn.importPublicKey(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "PublicKey",
			Message:  "Public key found",
		})
	}

	if found, err = cm.conn.isStackExists(cm.namer.GetClusterVPC()); err != nil {
		return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "VPC",
			Message:  "Not found, will be created new vpc",
		})
		if !dryRun {
			if err = cm.conn.createClusterVPC(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "VPC",
			Message:  fmt.Sprintf("Found vpc with id %v", cm.cluster.Status.Cloud.EKS.VpcId),
		})
	}

	if found, err = cm.conn.isControlPlaneExists(cm.cluster.Name); err != nil {
		return
	}
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Control plane",
			Message:  fmt.Sprintf("%v.compute.internal dscp option set will be created", cm.cluster.Spec.Config.Cloud.Region),
		})
		if !dryRun {
			if err = cm.conn.createControlPlane(); err != nil {
				return
			}
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Control plane",
			Message:  fmt.Sprintf("Found %v.compute.internal dscp option set", cm.cluster.Spec.Config.Cloud.Region),
		})
	}
	Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster)

	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}
	// wait for nodes to start
	if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
		return
	}

	cm.cluster.Status.Phase = api.ClusterReady
	if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
		return
	}

	return
}

func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	Logger(cm.ctx).Infoln("scaling node group...")
	var nodeGroups []*clusterapi.MachineSet
	nodeGroups, err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	var kc kubernetes.Interface
	kc, err = cm.GetAdminClient()
	if err != nil {
		return
	}
	for _, ng := range nodeGroups {
		igm := NewEKSNodeGroupManager(cm.ctx, cm.conn, ng, kc, cm.owner)
		var a2 []api.Action
		a2, err = igm.Apply(dryRun)
		if err != nil {
			return
		}
		acts = append(acts, a2...)
	}
	Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
	Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster)
	return
}

func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	Logger(cm.ctx).Infoln("deleting cluster...")
	if cm.cluster.Status.Phase == api.ClusterReady {
		cm.cluster.Status.Phase = api.ClusterDeleting
	}
	var found bool
	_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
	if err != nil {
		return
	}
	found, err = cm.conn.isControlPlaneExists(cm.cluster.Name)
	if err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if found {
		if err = cm.conn.deleteControlPlane(); err != nil {
			Logger(cm.ctx).Infof("Error on deleting control plane. Reason: %v", err)
		}
	}

	found, err = cm.conn.isStackExists(cm.namer.GetStackServiceRole())
	if err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if found {
		if err = cm.conn.deleteStack(cm.namer.GetStackServiceRole()); err != nil {
			Logger(cm.ctx).Infof("Error on deleting stack service role. Reason: %v", err)
		}
	}

	found, err = cm.conn.isStackExists(cm.namer.GetClusterVPC())
	if err != nil {
		return
	}
	if found {
		if err = cm.conn.deleteStack(cm.namer.GetClusterVPC()); err != nil {
			Logger(cm.ctx).Infof("Error on deleting cluster vpc. Reason: %v", err)
		}
	}

	found, err = cm.conn.getPublicKey()
	if err != nil {
		Logger(cm.ctx).Infoln(err)
	}
	if found {
		if err = cm.conn.deleteSSHKey(); err != nil {
			Logger(cm.ctx).Infof("Error on deleting SSH Key. Reason: %v", err)
		}
	}

	if !dryRun {
		cm.cluster.Status.Phase = api.ClusterDeleted
		Store(cm.ctx).Owner(cm.owner).Clusters().Update(cm.cluster)
	}

	return
}
