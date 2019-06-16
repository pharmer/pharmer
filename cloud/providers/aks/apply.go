package aks

import (
	"context"

	"github.com/pharmer/pharmer/apis/v1beta1/azure"

	containersvc "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2019-04-30/containerservice"
	"github.com/appscode/go/log"
	. "github.com/appscode/go/types"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/store"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cm *ClusterManager) PrepareCloud() error {
	err := cm.GetCloudConnector()
	if err != nil {
		return err
	}
	found, _ := cm.conn.getResourceGroup()
	if !found {
		if _, err := cm.conn.ensureResourceGroup(); err != nil {
			return err
		}
		log.Infof("Resource group %v in zone %v created", cm.namer.ResourceGroupName(), cm.Cluster.Spec.Config.Cloud.Zone)
	}
	nodeGroups, err := store.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(nodeGroups) > 1 {
		err = errors.Errorf("mutiple agent pool not supported yet")
		return err
	}
	agentPools := make([]containersvc.ManagedClusterAgentPoolProfile, 0)

	for _, ng := range nodeGroups {
		providerspec, err := azure.MachineSpecFromProviderSpec(ng.Spec.Template.Spec.ProviderSpec)
		if err != nil {
			return err
		}
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
		return err
	}

	return nil
}

func (cm *ClusterManager) ApplyScale() error {
	nodeGroups, err := store.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	cluster, err := cm.conn.managedClient.Get(context.Background(), cm.namer.ResourceGroupName(), cm.Cluster.Name)

	agentPools := make([]containersvc.ManagedClusterAgentPoolProfile, 0)
	for _, ng := range nodeGroups {
		providerspec, err := azure.MachineSpecFromProviderSpec(ng.Spec.Template.Spec.ProviderSpec)
		if err != nil {
			return err
		}
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
		}
		agentPools = append(agentPools, ap)
	}

	if len(agentPools) > 0 {
		if err = cm.conn.upsertAKS(agentPools); err != nil {
			return err
		}
	}
	return nil
}

func (cm *ClusterManager) applyUpgrade() error {
	if err := cm.conn.upgradeCluster(); err != nil {
		return err
	}
	cm.Cluster.Status.Phase = api.ClusterReady
	if _, err := store.StoreProvider.Clusters().UpdateStatus(cm.Cluster); err != nil {
		return err
	}
	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	if err := cm.conn.deleteAKS(); err != nil {
		return err
	}

	if err := cm.conn.deleteResourceGroup(); err != nil {
		return err
	}
	return nil
}
