package gke

import (
	"context"

	"github.com/appscode/go/wait"
	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/apis/v1beta1/gce"
	"pharmer.dev/pharmer/cloud"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	ProviderName = "gke"
)

type cloudConnector struct {
	*cloud.Scope
	containerService *container.Service
	computeService   *compute.Service
}

func newconnector(cm *ClusterManager) (*cloudConnector, error) {
	log := cm.Logger

	cluster := cm.Cluster
	cred, err := cm.StoreProvider.Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		log.Error(err, "failed to get credential from store")
		return nil, err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		log.Error(err, "credential is invalid", "credential-name", cluster.Spec.Config.CredentialName)
		return nil, err
	}

	cluster.Spec.Config.Cloud.Project = typed.ProjectID()

	serviceOpt := option.WithCredentialsJSON([]byte(typed.ServiceAccount()))
	containerService, err := container.NewService(context.Background(), serviceOpt)
	if err != nil {
		log.Error(err, "failed to create new container service")
		return nil, err
	}

	computeService, err := compute.NewService(context.Background(), serviceOpt)
	if err != nil {
		log.Error(err, "failed to create new compute service")
		return nil, err
	}

	conn := cloudConnector{
		Scope:            cm.Scope,
		containerService: containerService,
		computeService:   computeService,
	}
	if ok, msg := conn.IsUnauthorized(typed.ProjectID()); !ok {
		err = errors.New(msg)
		log.Error(err, "credential does not have necessary authorization", "credential-name", cluster.Spec.Config.CredentialName)
		return nil, err
	}
	return &conn, nil
}

// Returns true if unauthorized

func (conn *cloudConnector) IsUnauthorized(project string) (bool, string) {
	_, err := conn.containerService.Projects.Zones.Clusters.List(project, "us-central1-b").Do()
	if err != nil {
		return false, "Credential missing required authorization"
	}
	return true, ""
}

func (conn *cloudConnector) waitForZoneOperation(operation string) error {
	log := conn.Logger
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.containerService.Projects.Zones.Operations.Get(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, operation).Do()
		if err != nil {
			return false, nil
		}

		log.Info("Waiting for zonal operation", "attempt", attempt, "operation", operation, "status", r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) ensureNetworks() error {
	log := conn.Logger.WithValues("network-name", defaultNetwork)

	log.Info("Retrieving network", "project-id", conn.Cluster.Spec.Config.Cloud.Project)
	r2, err := conn.computeService.Networks.Insert(conn.Cluster.Spec.Config.Cloud.Project, &compute.Network{
		IPv4Range: conn.Cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		Name:      defaultNetwork,
	}).Do()
	log.V(2).Info("Retrieve network result", "response", r2)
	if err != nil {
		log.Error(err, "error creating network")
		return err
	}

	log.Info("New network created")

	return nil
}

func (conn *cloudConnector) getNetworks() (bool, error) {
	log := conn.Logger
	log.Info("Retrieving network", "network-name", defaultNetwork, "project-id", conn.Cluster.Spec.Config.Cloud.Project)
	r1, err := conn.computeService.Networks.Get(conn.Cluster.Spec.Config.Cloud.Project, defaultNetwork).Do()
	log.V(2).Info("Retrieve network result", "response", r1)
	if err != nil {
		log.Error(err, "error retrieving network")
		return false, err
	}
	conn.Cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{r1.IPv4Range}
	return true, nil
}

func (conn *cloudConnector) createCluster(cluster *container.Cluster) (string, error) {
	log := conn.Logger
	clusterRequest := &container.CreateClusterRequest{
		Cluster: cluster,
	}

	resp, err := conn.containerService.Projects.Zones.Clusters.Create(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, clusterRequest).Do()
	log.V(2).Info("Created kubernetes cluster", "response", resp)
	if err != nil {
		log.Error(err, "error creating cluster")
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) deleteCluster() (string, error) {
	log := conn.Logger
	resp, err := conn.containerService.Projects.Zones.Clusters.Delete(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, conn.Cluster.Name).Do()
	log.V(2).Info("Deleted kubernetes cluster", "response", resp)
	if err != nil {
		log.Error(err, "error deleting cluster")
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) addNodePool(ng *clusterapi.MachineSet) (string, error) {
	log := conn.Logger
	providerSpec, err := gce.MachineConfigFromProviderSpec(ng.Spec.Template.Spec.ProviderSpec)
	if err != nil {
		log.Error(err, "failed to get provider spec")
		return "", err
	}
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.Create(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, conn.Cluster.Name, &container.CreateNodePoolRequest{
		NodePool: &container.NodePool{
			Config: &container.NodeConfig{
				MachineType: providerSpec.MachineType,
				DiskSizeGb:  providerSpec.Disks[0].InitializeParams.DiskSizeGb,
				ImageType:   providerSpec.OS,
			},
			InitialNodeCount: int64(*ng.Spec.Replicas),
			Name:             ng.Name,
		},
	}).Do()
	if err != nil {
		log.Error(err, "failed to create nodepools", "nodepool-name", ng.Name)
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) deleteNoodPool(ng *clusterapi.MachineSet) (string, error) {
	log := conn.Logger

	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.Delete(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, conn.Cluster.Name, ng.Name).Do()
	log.V(2).Info("delete node pool", "response", resp)
	if err != nil {
		log.Error(err, "error deleting node pool")
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) adjustNodePool(ng *clusterapi.MachineSet) (string, error) {
	log := conn.Logger

	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.SetSize(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, conn.Cluster.Name, ng.Name,
		&container.SetNodePoolSizeRequest{
			NodeCount: int64(*ng.Spec.Replicas),
		}).Do()
	log.V(2).Info("adjust node pool", "response", resp)
	if err != nil {
		log.Error(err, "error adjusting node pool")
		return "", err
	}
	return resp.Name, nil
}
