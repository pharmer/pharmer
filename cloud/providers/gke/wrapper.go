package gke

import (
	"context"
	"crypto/rsa"
	"encoding/base64"

	. "github.com/appscode/go/context"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"google.golang.org/api/container/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func encodeCluster(ctx context.Context, cluster *api.Cluster, owner string) (*container.Cluster, error) {
	nodeGroups, err := Store(ctx).Owner(owner).MachineSet(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, ID(ctx))
		return nil, err
	}

	//pubKey := string(SSHKey(ctx).PublicKey)
	//value := fmt.Sprintf("%v:%v %v", cluster.Spec.Cloud.GKE.UserName, pubKey, cluster.Spec.Cloud.GKE.UserName)

	nodePools := make([]*container.NodePool, 0)
	for _, node := range nodeGroups {
		providerSpec := cluster.GKEProviderConfig(node.Spec.Template.Spec.ProviderSpec.Value.Raw)
		np := &container.NodePool{
			Config: &container.NodeConfig{
				MachineType: providerSpec.MachineType,
				DiskSizeGb:  providerSpec.Disks[0].InitializeParams.DiskSizeGb,
				ImageType:   providerSpec.OS,
				//		Tags:        cluster.Spec.Cloud.GKE.NodeTags,
				/*Metadata: map[string]string{

				},*/
			},
			InitialNodeCount: int64(*node.Spec.Replicas),
			Name:             node.Name,
		}
		nodePools = append(nodePools, np)
	}
	config := cluster.Spec.Config
	clusterApi := cluster.Spec.ClusterAPI

	kluster := &container.Cluster{
		ClusterIpv4Cidr:       clusterApi.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		Name:                  cluster.Name,
		InitialClusterVersion: cluster.Spec.Config.KubernetesVersion,

		MasterAuth: &container.MasterAuth{
			Username: config.Cloud.GKE.UserName,
			Password: config.Cloud.GKE.Password,
		},
		Network: config.Cloud.GKE.NetworkName,
		NetworkPolicy: &container.NetworkPolicy{
			Enabled:  true,
			Provider: config.Cloud.NetworkProvider,
		},
		NodePools: nodePools,
	}

	return kluster, nil
}

func (cm *ClusterManager) retrieveClusterStatus(cluster *container.Cluster) error {
	cm.cluster.Spec.ClusterAPI.Status.APIEndpoints = append(cm.cluster.Spec.ClusterAPI.Status.APIEndpoints, clusterapi.APIEndpoint{
		Host: cluster.Endpoint,
		Port: 0,
	})
	return nil
}

func (cm *ClusterManager) StoreCertificate(cluster *container.Cluster, owner string) error {
	certStore := Store(cm.ctx).Owner(owner).Certificates(cluster.Name)
	config := cm.cluster.Spec.Config
	_, caKey, err := certStore.Get(config.CACertName)
	if err == nil {
		if err = certStore.Delete(config.CACertName); err != nil {
			return err
		}
	}
	caCert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return err
	}
	crt, err := cert.ParseCertsPEM(caCert)
	if err != nil {
		return err
	}

	if err := certStore.Create(config.CACertName, crt[0], caKey); err != nil {
		return err
	}

	adminCert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientCertificate)
	if err != nil {
		return err
	}
	adminKey, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientKey)
	if err != nil {
		return err
	}

	aCrt, err := cert.ParseCertsPEM(adminCert)
	if err != nil {
		return err
	}

	aKey, err := cert.ParsePrivateKeyPEM(adminKey)
	if err != nil {
		return err
	}
	err = certStore.Create("admin", aCrt[0], aKey.(*rsa.PrivateKey))
	return err
}
