package gke

import (
	"context"

	. "github.com/appscode/go/context"
	"github.com/appscode/go/wait"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
)

const (
	ProviderName = "gke"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster

	containerService *container.Service
	computeService   *compute.Service
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.CredentialName)
	}

	cluster.Spec.Cloud.Project = typed.ProjectID()
	conf, err := google.JWTConfigFromJSON([]byte(typed.ServiceAccount()),
		container.CloudPlatformScope)
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}
	client := conf.Client(context.Background())
	containerService, err := container.New(client)
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}

	computeService, err := compute.New(client)
	if err != nil {
		return nil, errors.Wrap(err, ID(ctx))
	}

	conn := cloudConnector{
		ctx:              ctx,
		cluster:          cluster,
		containerService: containerService,
		computeService:   computeService,
	}
	if ok, msg := conn.IsUnauthorized(typed.ProjectID()); !ok {
		return nil, errors.Errorf("credential %s does not have necessary authorization. Reason: %s", cluster.Spec.CredentialName, msg)
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
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.containerService.Projects.Zones.Operations.Get(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Zone, operation).Do()
		if err != nil {
			return false, nil
		}

		Logger(conn.ctx).Infof("Attempt %v: Operation %v is %v ...", attempt, operation, r1.Status)
		if r1.Status == "DONE" {
			return true, nil
		}
		return false, nil
	})
}

func (conn *cloudConnector) ensureNetworks() error {
	Logger(conn.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, conn.cluster.Spec.Cloud.Project)
	r2, err := conn.computeService.Networks.Insert(conn.cluster.Spec.Cloud.Project, &compute.Network{
		IPv4Range: conn.cluster.Spec.Networking.PodSubnet,
		Name:      defaultNetwork,
	}).Do()
	Logger(conn.ctx).Debug("Created new network", r2, err)
	if err != nil {
		return errors.Wrap(err, ID(conn.ctx))
	}
	Logger(conn.ctx).Infof("New network %v is created", defaultNetwork)

	return nil
}

func (conn *cloudConnector) getNetworks() (bool, error) {
	Logger(conn.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, conn.cluster.Spec.Cloud.Project)
	r1, err := conn.computeService.Networks.Get(conn.cluster.Spec.Cloud.Project, defaultNetwork).Do()
	Logger(conn.ctx).Debug("Retrieve network result", r1, err)
	if err != nil {
		return false, err
	}
	conn.cluster.Spec.Networking.PodSubnet = r1.IPv4Range
	return true, nil
}

func (conn *cloudConnector) createCluster(cluster *container.Cluster) (string, error) {
	clusterRequest := &container.CreateClusterRequest{
		Cluster: cluster,
	}

	resp, err := conn.containerService.Projects.Zones.Clusters.Create(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Zone, clusterRequest).Do()
	Logger(conn.ctx).Debug("Created kubernetes cluster", resp, err)
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) deleteCluster() (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.Delete(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Zone, conn.cluster.Name).Do()
	Logger(conn.ctx).Debug("Deleted kubernetes cluster", resp, err)
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) addNodePool(ng *api.NodeGroup) (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.Create(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Zone, conn.cluster.Name, &container.CreateNodePoolRequest{
		NodePool: &container.NodePool{
			Config: &container.NodeConfig{
				MachineType: ng.Spec.Template.Spec.SKU,
				DiskSizeGb:  ng.Spec.Template.Spec.DiskSize,
				ImageType:   conn.cluster.Spec.Cloud.InstanceImage,
			},
			InitialNodeCount: ng.Spec.Nodes,
			Name:             ng.Name,
		},
	}).Do()
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) deleteNoodPool(ng *api.NodeGroup) (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.Delete(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Zone, conn.cluster.Name, ng.Name).Do()
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) adjustNoodPool(ng *api.NodeGroup) (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.SetSize(conn.cluster.Spec.Cloud.Project, conn.cluster.Spec.Cloud.Zone, conn.cluster.Name, ng.Name,
		&container.SetNodePoolSizeRequest{
			NodeCount: ng.Spec.Nodes,
		}).Do()
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}
