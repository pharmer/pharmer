package gke

import (
	"context"
	"crypto/rsa"
	"encoding/base64"

	. "github.com/appscode/go/context"
	api "github.com/pharmer/pharmer/apis/v1alpha1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pkg/errors"
	"google.golang.org/api/container/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
)

func encodeCluster(ctx context.Context, cluster *api.Cluster, owner string) (*container.Cluster, error) {
	nodeGroups, err := Store(ctx).Owner(owner).NodeGroups(cluster.Name).List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, ID(ctx))
		return nil, err
	}

	//pubKey := string(SSHKey(ctx).PublicKey)
	//value := fmt.Sprintf("%v:%v %v", cluster.Spec.Cloud.GKE.UserName, pubKey, cluster.Spec.Cloud.GKE.UserName)

	nodePools := make([]*container.NodePool, 0)
	for _, node := range nodeGroups {
		np := &container.NodePool{
			Config: &container.NodeConfig{
				MachineType: node.Spec.Template.Spec.SKU,
				DiskSizeGb:  node.Spec.Template.Spec.DiskSize,
				ImageType:   cluster.Spec.Cloud.InstanceImage,
				//		Tags:        cluster.Spec.Cloud.GKE.NodeTags,
				/*Metadata: map[string]string{

				},*/
			},
			InitialNodeCount: node.Spec.Nodes,
			Name:             node.Name,
		}
		nodePools = append(nodePools, np)
	}

	kluster := &container.Cluster{
		ClusterIpv4Cidr:       cluster.Spec.Networking.PodSubnet,
		Name:                  cluster.Name,
		InitialClusterVersion: cluster.Spec.KubernetesVersion,

		MasterAuth: &container.MasterAuth{
			Username: cluster.Spec.Cloud.GKE.UserName,
			Password: cluster.Spec.Cloud.GKE.Password,
		},
		Network: cluster.Spec.Cloud.GKE.NetworkName,
		NetworkPolicy: &container.NetworkPolicy{
			Enabled:  true,
			Provider: cluster.Spec.Networking.NetworkProvider,
		},
		NodePools: nodePools,
	}

	return kluster, nil
}

func (cm *ClusterManager) retrieveClusterStatus(cluster *container.Cluster) {
	cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
		Type:    core.NodeExternalIP,
		Address: cluster.Endpoint,
	})
}

func (cm *ClusterManager) StoreCertificate(cluster *container.Cluster) error {
	certStore := Store(cm.ctx).Certificates(cluster.Name)
	_, caKey, err := certStore.Get(cm.cluster.Spec.CACertName)
	if err == nil {
		certStore.Delete(cm.cluster.Spec.CACertName)
	}
	caCert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return err
	}
	crt, err := cert.ParseCertsPEM(caCert)
	if err != nil {
		return err
	}

	if err := certStore.Create(cm.cluster.Spec.CACertName, crt[0], caKey); err != nil {
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
