package framework

import (
	api "github.com/appscode/pharmer/apis/v1alpha1"
	"context"
	"github.com/appscode/pharmer/cloud"
	"fmt"
	_env "github.com/appscode/go/env"
)

const provider ="digitalocean"

func (c *clusterInvocation) GetName() string{
	return "storage-test"
}

func (c *clusterInvocation) GetSkeleton() (*api.Cluster, error) {
	fmt.Println(c.Config,"************", _env.Dev)
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
	_, err := c.Storage.Clusters().Update(cluster)
	return err
}

func (c *clusterInvocation) UpdateStatus(cluster *api.Cluster) error {
	cluster.Status.Phase = api.ClusterReady
	_, err := c.Storage.Clusters().Update(cluster)
	return err
}
