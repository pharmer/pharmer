package framework

import (
	"context"
	"fmt"

	_env "github.com/appscode/go/env"
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"github.com/appscode/pharmer/cloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const provider = "digitalocean"

func (c *clusterInvocation) GetName() string {
	return c.ClusterName
}

func (c *clusterInvocation) GetSkeleton() (*api.Cluster, error) {
	ctx := cloud.NewContext(context.Background(), c.Config, _env.Dev)

	cm, err := cloud.GetCloudManager(provider, ctx)
	if err != nil {
		return nil, err
	}
	cluster := &api.Cluster{}
	cluster.Name = c.GetName()
	cluster.Spec.Cloud.CloudProvider = "digitalocean"
	cluster.Spec.Cloud.Zone = "nyc3"
	cluster.Spec.CredentialName = "do"
	cluster.Spec.KubernetesVersion = "v1.8.0"
	fmt.Println(cm)
	//err = cm.SetDefaults(cluster)
	return cluster, err
}

func (c *clusterInvocation) Update(cluster *api.Cluster) error {
	cluster.Spec.KubernetesVersion = "v1.8.1"
	_, err := c.Storage.Clusters().Update(cluster)
	return err
}

func (c *clusterInvocation) CheckUpdate(cluster *api.Cluster) error {
	if cluster.Spec.KubernetesVersion == "v1.8.1" {
		return nil
	}
	return fmt.Errorf("cluster was not updated")
}

func (c *clusterInvocation) UpdateStatus(cluster *api.Cluster) error {
	cluster.Status.Phase = api.ClusterReady
	_, err := c.Storage.Clusters().Update(cluster)
	return err
}

func (c *clusterInvocation) CheckUpdateStatus(cluster *api.Cluster) error {
	if cluster.Status.Phase == api.ClusterReady {
		return nil
	}
	return fmt.Errorf("cluster status was not updated")
}
func (c *clusterInvocation) List() error {
	clusters, err := c.Storage.Clusters().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(clusters) < 1 {
		return fmt.Errorf("can't list clusters")
	}
	return nil
}
