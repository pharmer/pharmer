package gke

import (
	"context"

	"github.com/appscode/go/log"
	"github.com/appscode/go/wait"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"github.com/pharmer/pharmer/apis/v1beta1/gce"
	"github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
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
	cluster := cm.Cluster
	cred, err := cm.StoreProvider.Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.Config.CredentialName)
	}

	cluster.Spec.Config.Cloud.Project = typed.ProjectID()

	serviceOpt := option.WithCredentialsJSON([]byte(typed.ServiceAccount()))
	containerService, err := container.NewService(context.Background(), serviceOpt)
	if err != nil {
		return nil, err
	}

	computeService, err := compute.NewService(context.Background(), serviceOpt)
	if err != nil {
		return nil, err
	}

	conn := cloudConnector{
		Scope:            cm.Scope,
		containerService: containerService,
		computeService:   computeService,
	}
	if ok, msg := conn.IsUnauthorized(typed.ProjectID()); !ok {
		return nil, errors.Errorf("credential %s does not have necessary authorization. Reason: %s", cluster.Spec.Config.CredentialName, msg)
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
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.containerService.Projects.Zones.Operations.Get(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, operation).Do()
		if err != nil {
			return false, nil
		}

		log.Infof("Attempt %v: Operation %v is %v ...", attempt, operation, r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) ensureNetworks() error {
	log.Infof("Retrieving network %v for project %v", defaultNetwork, conn.Cluster.Spec.Config.Cloud.Project)
	r2, err := conn.computeService.Networks.Insert(conn.Cluster.Spec.Config.Cloud.Project, &compute.Network{
		IPv4Range: conn.Cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		Name:      defaultNetwork,
	}).Do()
	log.Debug("Created new network", r2, err)
	if err != nil {
		return err
	}
	log.Infof("New network %v is created", defaultNetwork)

	return nil
}

func (conn *cloudConnector) getNetworks() (bool, error) {
	log.Infof("Retrieving network %v for project %v", defaultNetwork, conn.Cluster.Spec.Config.Cloud.Project)
	r1, err := conn.computeService.Networks.Get(conn.Cluster.Spec.Config.Cloud.Project, defaultNetwork).Do()
	log.Debug("Retrieve network result", r1, err)
	if err != nil {
		return false, err
	}
	conn.Cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{r1.IPv4Range}
	return true, nil
}

func (conn *cloudConnector) createCluster(cluster *container.Cluster) (string, error) {
	clusterRequest := &container.CreateClusterRequest{
		Cluster: cluster,
	}

	resp, err := conn.containerService.Projects.Zones.Clusters.Create(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, clusterRequest).Do()
	log.Debug("Created kubernetes cluster", resp, err)
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) deleteCluster() (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.Delete(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, conn.Cluster.Name).Do()
	log.Debug("Deleted kubernetes cluster", resp, err)
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) addNodePool(ng *clusterapi.MachineSet) (string, error) {
	providerSpec, err := gce.MachineConfigFromProviderSpec(ng.Spec.Template.Spec.ProviderSpec)
	if err != nil {
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
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) deleteNoodPool(ng *clusterapi.MachineSet) (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.Delete(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, conn.Cluster.Name, ng.Name).Do()
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) adjustNoodPool(ng *clusterapi.MachineSet) (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.SetSize(conn.Cluster.Spec.Config.Cloud.Project, conn.Cluster.Spec.Config.Cloud.Zone, conn.Cluster.Name, ng.Name,
		&container.SetNodePoolSizeRequest{
			NodeCount: int64(*ng.Spec.Replicas),
		}).Do()
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}
