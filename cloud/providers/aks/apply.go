package aks

import (
	"context"

	containersvc "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2019-06-01/containerservice"
	"github.com/appscode/go/types"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pharmer.dev/pharmer/apis/v1alpha1/azure"
)

func (cm *ClusterManager) PrepareCloud() error {
	log := cm.Logger
	log.Info("Preparing cloud infra")

	err := cm.SetCloudConnector()
	if err != nil {
		log.Error(err, "failed to set aks cloud connector")
		return err
	}
	found, _ := cm.conn.getResourceGroup()
	if !found {
		if _, err := cm.conn.ensureResourceGroup(); err != nil {
			log.Error(err, "failed to ensure resource group")
			return err
		}
		log.Info("Resource group created", "resourcegroup-name", cm.namer.ResourceGroupName(), "zone", cm.Cluster.Spec.Config.Cloud.Zone)
	}
	nodeGroups, err := cm.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list machineset from store")
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
			log.Error(err, "failed to get provider spec")
			return err
		}
		name := cm.namer.GetNodeGroupName(ng.Name)
		ap := containersvc.ManagedClusterAgentPoolProfile{
			Name:   types.StringP(name),
			Count:  ng.Spec.Replicas,
			VMSize: containersvc.VMSizeTypes(providerspec.VMSize),
			OsType: containersvc.OSType(providerspec.OSDisk.OSType),
		}
		agentPools = append(agentPools, ap)
	}
	if err = cm.conn.upsertAKS(agentPools); err != nil {
		log.Error(err, "failed to upsert nodepools")
		return err
	}

	return nil
}

func (cm *ClusterManager) ApplyScale() error {
	log := cm.Logger
	nodeGroups, err := cm.StoreProvider.MachineSet(cm.Cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list machineset from store")
		return err
	}

	cluster, err := cm.conn.managedClient.Get(context.Background(), cm.namer.ResourceGroupName(), cm.Cluster.Name)
	if err != nil {
		log.Error(err, "failed to get aks cluster")
		return err
	}

	agentPools := make([]containersvc.ManagedClusterAgentPoolProfile, 0)
	for _, ng := range nodeGroups {
		providerspec, err := azure.MachineSpecFromProviderSpec(ng.Spec.Template.Spec.ProviderSpec)
		if err != nil {
			log.Error(err, "failed to get provider spec")
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
			Name:   types.StringP(name),
			Count:  ng.Spec.Replicas,
			VMSize: containersvc.VMSizeTypes(providerspec.VMSize),
			OsType: containersvc.Linux,
		}
		agentPools = append(agentPools, ap)
	}

	if len(agentPools) > 0 {
		if err = cm.conn.upsertAKS(agentPools); err != nil {
			log.Error(err, "failed to upsert nodepools")
			return err
		}
	}
	return nil
}

func (cm *ClusterManager) ApplyDelete() error {
	log := cm.Logger
	if err := cm.conn.deleteAKS(); err != nil {
		log.Error(err, "failed to delete cluster")
		return err
	}

	if err := cm.conn.deleteResourceGroup(); err != nil {
		log.Error(err, "failed to delete resource group")
		return err
	}
	return nil
}
