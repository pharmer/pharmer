package aks

import (
	"context"

	containersvc "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2019-02-01/containerservice"
	. "github.com/appscode/go/context"
	. "github.com/appscode/go/types"
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
	if cm.ctx, err = LoadSSHKey(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	if cm.conn, err = NewConnector(cm.ctx, cm.cluster, cm.owner); err != nil {
		return nil, err
	}
	cm.conn.namer = cm.namer

	if cm.cluster.Status.Phase == api.ClusterUpgrading {
		return nil, errors.Errorf("cluster `%s` is upgrading. Retry after cluster returns to Ready state", cm.cluster.Name)
	}
	if cm.cluster.Status.Phase == api.ClusterReady {
		if upgrade, err := cm.conn.getUpgradeProfile(); err != nil {
			return nil, err
		} else if upgrade {
			cm.cluster.Status.Phase = api.ClusterUpgrading
			Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
			return cm.applyUpgrade(dryRun)
		}
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

// IP >>>>>>>>>>>>>>>>
// TODO(tamal): if cluster.Spec.ctx.MasterReservedIP == "auto"
//	name := cluster.Spec.ctx.KubernetesMasterName + "-pip"
//	// cluster.Spec.ctx.MasterExternalIP = *ip.IPAddress
//	cluster.Spec.ctx.MasterReservedIP = *ip.IPAddress
//	// cluster.Spec.ctx.ApiServerUrl = "https://" + *ip.IPAddress

func (cm *ClusterManager) applyCreate(dryRun bool) (acts []api.Action, err error) {
	var found bool
	found, _ = cm.conn.getResourceGroup()
	if !found {
		acts = append(acts, api.Action{
			Action:   api.ActionAdd,
			Resource: "Resource group",
			Message:  "Resource group will be created",
		})
		if !dryRun {
			if _, err = cm.conn.ensureResourceGroup(); err != nil {
				return
			}
			Logger(cm.ctx).Infof("Resource group %v in zone %v created", cm.namer.ResourceGroupName(), cm.cluster.Spec.Config.Cloud.Zone)
		}
	} else {
		acts = append(acts, api.Action{
			Action:   api.ActionNOP,
			Resource: "Resource group",
			Message:  "Resource group found",
		})
	}
	if !dryRun {
		var nodeGroups []*clusterapi.MachineSet
		nodeGroups, err = Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
		if err != nil {
			err = errors.Wrap(err, ID(cm.ctx))
			return
		}
		if len(nodeGroups) > 1 {
			err = errors.Errorf("mutiple agent pool not supported yet")
			return
		}
		agentPools := make([]containersvc.ManagedClusterAgentPoolProfile, 0)

		for _, ng := range nodeGroups {
			providerspec := cm.cluster.AKSProviderConfig(ng.Spec.Template.Spec.ProviderSpec.Value.Raw)
			name := cm.namer.GetNodeGroupName(ng.Name)
			ap := containersvc.ManagedClusterAgentPoolProfile{
				Name:   StringP(name),
				Count:  ng.Spec.Replicas,
				VMSize: containersvc.VMSizeTypes(providerspec.VMSize),
				OsType: containersvc.OSType(providerspec.OSDisk.OSType),
			}
			agentPools = append(agentPools, ap)
		}
		if err = cm.conn.upsertAKS(agentPools); err != nil {
			return
		}
		var kc kubernetes.Interface
		kc, err = cm.GetAdminClient()
		if err != nil {
			return
		}
		// wait for nodes to start
		if err = WaitForReadyMaster(cm.ctx, kc); err != nil {
			//return
		}

		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}

	return
}

func (cm *ClusterManager) applyScale(dryRun bool) (acts []api.Action, err error) {
	nodeGroups, err := Store(cm.ctx).Owner(cm.owner).MachineSet(cm.cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, ID(cm.ctx))
		return
	}

	var cluster containersvc.ManagedCluster
	cluster, err = cm.conn.managedClient.Get(context.Background(), cm.namer.ResourceGroupName(), cm.cluster.Name)

	agentPools := make([]containersvc.ManagedClusterAgentPoolProfile, 0)
	for _, ng := range nodeGroups {
		providerspec := cm.cluster.AKSProviderConfig(ng.Spec.Template.Spec.ProviderSpec.Value.Raw)
		name := cm.namer.GetNodeGroupName(ng.Name)
		found := false
		for _, a := range *cluster.AgentPoolProfiles {
			if *a.Name == name {
				if *a.Count == *ng.Spec.Replicas {
					found = true
					break
				}
			}
		}
		if found {
			continue
		}
		ap := containersvc.ManagedClusterAgentPoolProfile{
			Name:   StringP(name),
			Count:  ng.Spec.Replicas,
			VMSize: containersvc.VMSizeTypes(providerspec.VMSize),
			OsType: containersvc.Linux,
			//DNSPrefix:    StringP(name),
			//Fqdn:         StringP(name),
			//VnetSubnetID: subnet.ID,
		}
		agentPools = append(agentPools, ap)
	}

	if len(agentPools) > 0 {
		if err = cm.conn.upsertAKS(agentPools); err != nil {
			return
		}
	}
	return
}

func (cm *ClusterManager) applyDelete(dryRun bool) (acts []api.Action, err error) {
	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Container cluster",
		Message:  "cluster will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteAKS(); err != nil {
			return
		}
	}

	acts = append(acts, api.Action{
		Action:   api.ActionDelete,
		Resource: "Resource group",
		Message:  "Resource group will be deleted",
	})
	if !dryRun {
		if err = cm.conn.deleteResourceGroup(); err != nil {
			return
		}
		// Failed
		cm.cluster.Status.Phase = api.ClusterDeleted
		_, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster)
		if err != nil {
			return
		}
	}
	return
}

func (cm *ClusterManager) applyUpgrade(dryRun bool) (acts []api.Action, err error) {
	if !dryRun {
		if err = cm.conn.upgradeCluster(); err != nil {
			return
		}
		cm.cluster.Status.Phase = api.ClusterReady
		if _, err = Store(cm.ctx).Owner(cm.owner).Clusters().UpdateStatus(cm.cluster); err != nil {
			return
		}
	}
	return
}
