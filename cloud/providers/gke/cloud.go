package gke

import (
	"context"

	. "github.com/appscode/go/context"
	"github.com/appscode/go/wait"
	"github.com/pharmer/cloud/pkg/credential"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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

func NewConnector(ctx context.Context, cluster *api.Cluster, owner string) (*cloudConnector, error) {
	cred, err := Store(ctx).Owner(owner).Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.GCE{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.Config.CredentialName)
	}

	cluster.Spec.Config.Cloud.Project = typed.ProjectID()
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
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++

		r1, err := conn.containerService.Projects.Zones.Operations.Get(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, operation).Do()
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
	Logger(conn.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, conn.cluster.Spec.Config.Cloud.Project)
	r2, err := conn.computeService.Networks.Insert(conn.cluster.Spec.Config.Cloud.Project, &compute.Network{
		IPv4Range: conn.cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
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
	Logger(conn.ctx).Infof("Retrieving network %v for project %v", defaultNetwork, conn.cluster.Spec.Config.Cloud.Project)
	r1, err := conn.computeService.Networks.Get(conn.cluster.Spec.Config.Cloud.Project, defaultNetwork).Do()
	Logger(conn.ctx).Debug("Retrieve network result", r1, err)
	if err != nil {
		return false, err
	}
	conn.cluster.Spec.ClusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{r1.IPv4Range}
	return true, nil
}

func (conn *cloudConnector) createCluster(cluster *container.Cluster) (string, error) {
	clusterRequest := &container.CreateClusterRequest{
		Cluster: cluster,
	}

	resp, err := conn.containerService.Projects.Zones.Clusters.Create(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, clusterRequest).Do()
	Logger(conn.ctx).Debug("Created kubernetes cluster", resp, err)
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) deleteCluster() (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, conn.cluster.Name).Do()
	Logger(conn.ctx).Debug("Deleted kubernetes cluster", resp, err)
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) addNodePool(ng *clusterapi.MachineSet) (string, error) {
	providerSpec := conn.cluster.GKEProviderConfig(ng.Spec.Template.Spec.ProviderSpec.Value.Raw)
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.Create(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, conn.cluster.Name, &container.CreateNodePoolRequest{
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
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.Delete(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, conn.cluster.Name, ng.Name).Do()
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

func (conn *cloudConnector) adjustNoodPool(ng *clusterapi.MachineSet) (string, error) {
	resp, err := conn.containerService.Projects.Zones.Clusters.NodePools.SetSize(conn.cluster.Spec.Config.Cloud.Project, conn.cluster.Spec.Config.Cloud.Zone, conn.cluster.Name, ng.Name,
		&container.SetNodePoolSizeRequest{
			NodeCount: int64(*ng.Spec.Replicas),
		}).Do()
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}
