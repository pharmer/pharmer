package dokube

import (
	"context"
	"strconv"
	"time"

	"github.com/appscode/go/wait"
	"github.com/digitalocean/godo"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/credential"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ProviderName = "dokube"
)

type cloudConnector struct {
	ctx     context.Context
	cluster *api.Cluster
	client  *godo.Client
}

func NewConnector(ctx context.Context, cluster *api.Cluster) (*cloudConnector, error) {
	cred, err := Store(ctx).Credentials().Get(cluster.Spec.CredentialName)
	if err != nil {
		return nil, err
	}
	typed := credential.DigitalOcean{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		return nil, errors.Wrapf(err, "credential %s is invalid", cluster.Spec.CredentialName)
	}
	oauthClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: typed.Token(),
	}))
	conn := cloudConnector{
		ctx:     ctx,
		cluster: cluster,
		client:  godo.NewClient(oauthClient),
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		return nil, errors.Errorf("credential `%s` does not have necessary authorization. Reason: %s", cluster.Spec.CredentialName, msg)
	}
	return &conn, nil
}

func (conn *cloudConnector) IsUnauthorized() (bool, string) {
	name := "check-write-access:" + strconv.FormatInt(time.Now().Unix(), 10)
	_, _, err := conn.client.Tags.Create(context.TODO(), &godo.TagCreateRequest{
		Name: name,
	})
	if err != nil {
		return false, "Credential missing WRITE scope"
	}
	_, err = conn.client.Tags.Delete(context.TODO(), name)
	if err != nil {
		return false, "Unable to delete tag"
	}
	return true, ""
}

func (conn *cloudConnector) createCluster(cluster *api.Cluster) (*godo.KubernetesCluster, error) {
	nodeGroups, err := Store(conn.ctx).NodeGroups(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodePools := make([]*godo.KubernetesNodePoolCreateRequest, len(nodeGroups))
	i := 0
	for _, node := range nodeGroups {
		nodePools[i] = &godo.KubernetesNodePoolCreateRequest{
			Name:  node.Name,
			Size:  node.Spec.Template.Spec.SKU,
			Count: int(node.Spec.Nodes),
		}
		i++
	}

	clusterCreateReq := &godo.KubernetesClusterCreateRequest{
		Name:        cluster.Name,
		RegionSlug:  cluster.Spec.Cloud.Zone,
		VersionSlug: cluster.Spec.KubernetesVersion,
		NodePools:   nodePools,
	}
	kubeCluster, _, err := conn.client.Kubernetes.Create(conn.ctx, clusterCreateReq)
	if err != nil {
		return nil, err
	}
	if err = conn.waitForClusterCreation(kubeCluster); err != nil {
		conn.cluster.Status.Reason = err.Error()
		return nil, err
	}
	kubeCluster, err = conn.getCluster(kubeCluster.ID)
	if err != nil {
		return nil, err
	}
	return kubeCluster, nil
}

func (conn *cloudConnector) getCluster(clusterID string) (*godo.KubernetesCluster, error) {
	cluster, _, err := conn.client.Kubernetes.Get(conn.ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (conn *cloudConnector) deleteCluster() error {
	_, err := conn.client.Kubernetes.Delete(conn.ctx, conn.cluster.Spec.Cloud.Dokube.ClusterID)
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) waitForClusterCreation(cluster *godo.KubernetesCluster) error {
	attempt := 0
	return wait.PollImmediate(RetryInterval, RetryTimeout, func() (bool, error) {
		attempt++
		cluster, _, _ := conn.client.Kubernetes.Get(conn.ctx, cluster.ID)
		Logger(conn.ctx).Infof("Attempt %v: Creating Cluster %v ...", attempt, cluster.Name)
		if cluster.Status.State == "provisioning" {
			return false, nil
		}
		return true, nil
	})

}

func (conn *cloudConnector) getNodePool(ng *api.NodeGroup) (*godo.KubernetesNodePool, error) {
	npID, err := conn.getNodePoolIDFromName(ng.Name)
	if err != nil {
		return nil, err
	}
	np, _, err := conn.client.Kubernetes.GetNodePool(conn.ctx, conn.cluster.Spec.Cloud.Dokube.ClusterID, npID)
	if err != nil {
		return nil, err
	}
	return np, nil
}

func (conn *cloudConnector) addNodePool(ng *api.NodeGroup) error {
	_, _, err := conn.client.Kubernetes.CreateNodePool(conn.ctx, conn.cluster.Spec.Cloud.Dokube.ClusterID, &godo.KubernetesNodePoolCreateRequest{
		Name:  ng.Name,
		Size:  ng.Spec.Template.Spec.SKU,
		Count: int(ng.Spec.Nodes),
	})
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) deleteNodePool(ng *api.NodeGroup) error {
	npID, err := conn.getNodePoolIDFromName(ng.Name)
	if err != nil {
		return err
	}
	_, err = conn.client.Kubernetes.DeleteNodePool(conn.ctx, conn.cluster.Spec.Cloud.Dokube.ClusterID, npID)
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) adjustNodePool(ng *api.NodeGroup) error {
	npID, err := conn.getNodePoolIDFromName(ng.Name)
	if err != nil {
		return err
	}
	_, _, err = conn.client.Kubernetes.UpdateNodePool(conn.ctx, conn.cluster.Spec.Cloud.Dokube.ClusterID, npID, &godo.KubernetesNodePoolUpdateRequest{
		Name:  ng.Name,
		Count: int(ng.Spec.Nodes),
	})
	if err != nil {
		return err
	}
	return nil
}

func (conn *cloudConnector) getNodePoolIDFromName(name string) (string, error) {
	knps, _, err := conn.client.Kubernetes.ListNodePools(conn.ctx, conn.cluster.Spec.Cloud.Dokube.ClusterID, &godo.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, knp := range knps {
		if knp.Name == name {
			return knp.ID, nil
		}
	}

	return "", errors.Errorf("NodePool with name %v not found!", name)
}
