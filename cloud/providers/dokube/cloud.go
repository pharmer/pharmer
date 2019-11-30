/*
Copyright The Pharmer Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package dokube

import (
	"context"
	"strconv"
	"time"

	"pharmer.dev/cloud/pkg/credential"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	dokube_config "pharmer.dev/pharmer/apis/v1alpha1/dokube"
	"pharmer.dev/pharmer/cloud"

	"github.com/appscode/go/wait"
	"github.com/digitalocean/godo"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	ProviderName = "dokube"
)

type cloudConnector struct {
	*cloud.Scope
	client *godo.Client
}

func newconnector(cm *ClusterManager) (*cloudConnector, error) {
	log := cm.Logger
	cluster := cm.Cluster
	cred, err := cm.StoreProvider.Credentials().Get(cluster.Spec.Config.CredentialName)
	if err != nil {
		log.Error(err, "failed to get credential from store")
		return nil, err
	}
	typed := credential.DigitalOcean{CommonSpec: credential.CommonSpec(cred.Spec)}
	if ok, err := typed.IsValid(); !ok {
		log.Error(err, "invalid credential", "cred-name", cluster.Spec.Config.CredentialName)
		return nil, err
	}
	oauthClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: typed.Token(),
	}))
	conn := cloudConnector{
		Scope:  cm.Scope,
		client: godo.NewClient(oauthClient),
	}
	if ok, msg := conn.IsUnauthorized(); !ok {
		err = errors.Errorf("credential `%s` does not have necessary authorization. Reason: %s", cluster.Spec.Config.CredentialName, msg)
		log.Error(err, "unauthorized credential")
		return nil, err
	}
	return &conn, nil
}

func (conn *cloudConnector) IsUnauthorized() (bool, string) {
	log := conn.Logger
	name := "check-write-access:" + strconv.FormatInt(time.Now().Unix(), 10)
	_, _, err := conn.client.Tags.Create(context.TODO(), &godo.TagCreateRequest{
		Name: name,
	})
	if err != nil {
		log.Error(err, "failed to create tags", "tag-name", name)
		return false, "Credential missing WRITE scope"
	}
	_, err = conn.client.Tags.Delete(context.TODO(), name)
	if err != nil {
		log.Error(err, "failed to delete tag", "tag-name", name)
		return false, "Unable to delete tag"
	}
	return true, ""
}

func (conn *cloudConnector) createCluster(cluster *api.Cluster) (*godo.KubernetesCluster, error) {
	log := conn.Logger
	nodeGroups, err := conn.StoreProvider.MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "failed to list machineset from store")
		return nil, err
	}
	nodePools := make([]*godo.KubernetesNodePoolCreateRequest, len(nodeGroups))

	i := 0
	for _, node := range nodeGroups {
		config, err := dokube_config.DokubeProviderConfig(node.Spec.Template.Spec.ProviderSpec.Value.Raw)
		if err != nil {
			log.Error(err, "failed to get provider config")
			return nil, err
		}
		nodePools[i] = &godo.KubernetesNodePoolCreateRequest{
			Name:  node.Name,
			Size:  config.Size,
			Count: int(*node.Spec.Replicas),
		}
		i++
	}

	clusterCreateReq := &godo.KubernetesClusterCreateRequest{
		Name:        cluster.Name,
		RegionSlug:  cluster.Spec.Config.Cloud.Zone,
		VersionSlug: cluster.Spec.Config.KubernetesVersion,
		NodePools:   nodePools,
	}
	kubeCluster, _, err := conn.client.Kubernetes.Create(context.Background(), clusterCreateReq)
	log.V(4).Info("creating cluster", "cluster-create-req", clusterCreateReq)
	if err != nil {
		log.Error(err, "failed to create cluster")
		return nil, err
	}
	if err = conn.waitForClusterCreation(kubeCluster); err != nil {
		log.Error(err, "failed to create cluster")
		conn.Cluster.Status.Reason = err.Error()
		return nil, err
	}
	kubeCluster, err = conn.getCluster(kubeCluster.ID)
	if err != nil {
		log.Error(err, "failed to get cluster", "id", kubeCluster.ID)
		return nil, err
	}
	return kubeCluster, nil
}

func (conn *cloudConnector) getCluster(clusterID string) (*godo.KubernetesCluster, error) {
	cluster, _, err := conn.client.Kubernetes.Get(context.Background(), clusterID)
	if err != nil {
		conn.Logger.Error(err, "failed to get digitalocean cluster")
		return nil, err
	}
	return cluster, nil
}

func (conn *cloudConnector) waitForClusterCreation(cluster *godo.KubernetesCluster) error {
	log := conn.Logger
	attempt := 0
	return wait.PollImmediate(api.RetryInterval, api.RetryTimeout, func() (bool, error) {
		attempt++
		cluster, _, _ := conn.client.Kubernetes.Get(context.Background(), cluster.ID)
		log.Info("Waiting for cluster creation", "attempt", attempt, "status", cluster.Status.State)
		if cluster.Status.State == "provisioning" {
			return false, nil
		}
		return true, nil
	})

}

func (conn *cloudConnector) getNodePool(ng *clusterapi.MachineSet) (*godo.KubernetesNodePool, error) {
	log := conn.Logger.WithValues("nodepool-name", ng.Name)
	npID, err := conn.getNodePoolIDFromName(ng.Name)
	if err != nil {
		log.Error(err, "failed to get node pool id from name")
		return nil, err
	}
	np, _, err := conn.client.Kubernetes.GetNodePool(context.Background(), conn.Cluster.Spec.Config.Cloud.Dokube.ClusterID, npID)
	if err != nil {
		log.Error(err, "failed to get nodepool")
		return nil, err
	}
	return np, nil
}

func (conn *cloudConnector) addNodePool(ng *clusterapi.MachineSet) error {
	log := conn.Logger.WithValues("nodepool-name", ng.Name)
	config, err := dokube_config.DokubeProviderConfig(ng.Spec.Template.Spec.ProviderSpec.Value.Raw)
	if err != nil {
		log.Error(err, "failed to get provider config")
		return err
	}
	_, _, err = conn.client.Kubernetes.CreateNodePool(context.Background(), conn.Cluster.Spec.Config.Cloud.Dokube.ClusterID, &godo.KubernetesNodePoolCreateRequest{
		Name:  ng.Name,
		Size:  config.Size,
		Count: int(*ng.Spec.Replicas),
	})
	if err != nil {
		log.Error(err, "failed to create nodepool")
		return err
	}
	return nil
}

func (conn *cloudConnector) deleteNodePool(ng *clusterapi.MachineSet) error {
	log := conn.Logger.WithValues("nodepool-name", ng.Name)
	npID, err := conn.getNodePoolIDFromName(ng.Name)
	if err != nil {
		log.Error(err, "failed to get nodepool id from name")
		return err
	}
	_, err = conn.client.Kubernetes.DeleteNodePool(context.Background(), conn.Cluster.Spec.Config.Cloud.Dokube.ClusterID, npID)
	if err != nil {
		log.Error(err, "failed to delete nodepool")
		return err
	}
	return nil
}

func (conn *cloudConnector) adjustNodePool(ng *clusterapi.MachineSet) error {
	log := conn.Logger.WithValues("nodepool-name", ng.Name)
	npID, err := conn.getNodePoolIDFromName(ng.Name)
	if err != nil {
		log.Error(err, "failed to get nodepool id from name")
		return err
	}
	count := int(*ng.Spec.Replicas)
	_, _, err = conn.client.Kubernetes.UpdateNodePool(context.Background(), conn.Cluster.Spec.Config.Cloud.Dokube.ClusterID, npID, &godo.KubernetesNodePoolUpdateRequest{
		Name:  ng.Name,
		Count: &count,
	})
	if err != nil {
		log.Error(err, "failed to update node pool")
		return err
	}
	return nil
}

func (conn *cloudConnector) getNodePoolIDFromName(name string) (string, error) {
	knps, _, err := conn.client.Kubernetes.ListNodePools(context.Background(), conn.Cluster.Spec.Config.Cloud.Dokube.ClusterID, &godo.ListOptions{})
	if err != nil {
		conn.Logger.Error(err, "failed to list nodepools")
		return "", err
	}

	for _, knp := range knps {
		if knp.Name == name {
			return knp.ID, nil
		}
	}

	return "", errors.Errorf("NodePool with name %v not found!", name)
}
