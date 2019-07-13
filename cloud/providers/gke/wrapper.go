package gke

import (
	"crypto/rsa"
	"encoding/base64"

	"gomodules.xyz/cert"
	"google.golang.org/api/container/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "pharmer.dev/pharmer/apis/v1beta1"
	"pharmer.dev/pharmer/apis/v1beta1/gce"
	"pharmer.dev/pharmer/store"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func encodeCluster(machinesetStore store.MachineSetStore, cluster *api.Cluster) (*container.Cluster, error) {
	nodeGroups, err := machinesetStore.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	//pubKey := string(SSHKey(ctx).PublicKey)
	//value := fmt.Sprintf("%v:%v %v", cluster.Spec.Cloud.GKE.UserName, pubKey, cluster.Spec.Cloud.GKE.UserName)

	nodePools := make([]*container.NodePool, 0)
	for _, node := range nodeGroups {
		providerSpec, err := gce.MachineConfigFromProviderSpec(node.Spec.Template.Spec.ProviderSpec)
		if err != nil {
			return nil, err
		}
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
	clusterAPI := cluster.Spec.ClusterAPI

	kluster := &container.Cluster{
		ClusterIpv4Cidr:       clusterAPI.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		Name:                  cluster.Name,
		InitialClusterVersion: cluster.Spec.Config.KubernetesVersion,

		MasterAuth: &container.MasterAuth{
			Username: config.Cloud.GKE.UserName,
			Password: config.Cloud.GKE.Password,
			ClientCertificateConfig: &container.ClientCertificateConfig{
				IssueClientCertificate: true,
			},
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
	cm.Cluster.Spec.ClusterAPI.Status.APIEndpoints = append(cm.Cluster.Spec.ClusterAPI.Status.APIEndpoints, clusterapi.APIEndpoint{
		Host: cluster.Endpoint,
		Port: 0,
	})
	return nil
}

func (cm *ClusterManager) StoreCertificate(certStore store.CertificateStore, cluster *container.Cluster) error {
	log := cm.Logger

	_, caKey, err := certStore.Get(api.CACertName)
	if err == nil {
		if err = certStore.Delete(api.CACertName); err != nil {
			log.Error(err, "failed to delete ca-cert from store")
			return err
		}
	}
	caCert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		log.Error(err, "failed to base64 decode cluster ca-cert")
		return err
	}
	crt, err := cert.ParseCertsPEM(caCert)
	if err != nil {
		log.Error(err, "failed to parse cert pem")
		return err
	}

	if err := certStore.Create(api.CACertName, crt[0], caKey); err != nil {
		log.Error(err, "failed to create ca-cert in store")
		return err
	}
	cm.Certs.CACert.Cert = crt[0]

	adminCert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientCertificate)
	if err != nil {
		log.Error(err, "failed to base64 decode cluster admin-cert")
		return err
	}
	adminKey, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientKey)
	if err != nil {
		log.Error(err, "failed to base64 decode cluster admin-key")
		return err
	}

	aCrt, err := cert.ParseCertsPEM(adminCert)
	if err != nil {
		log.Error(err, "failed to parse admin-cert PEM")
		return err
	}

	aKey, err := cert.ParsePrivateKeyPEM(adminKey)
	if err != nil {
		log.Error(err, "failed to parse admin-key PEM")
		return err
	}

	err = certStore.Create("admin", aCrt[0], aKey.(*rsa.PrivateKey))
	if err != nil {
		log.Error(err, "failed to create admin-cert in store")
		return err
	}

	return nil
}
