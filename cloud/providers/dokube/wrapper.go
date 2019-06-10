package dokube

import (
	"context"
	"crypto/rsa"
	"net/url"

	"github.com/digitalocean/godo"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	. "github.com/pharmer/pharmer/cloud"
	"github.com/pharmer/pharmer/store"
	"gomodules.xyz/cert"
	"k8s.io/client-go/tools/clientcmd"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) retrieveClusterStatus(cluster *godo.KubernetesCluster) error {
	u, err := url.Parse(cluster.Endpoint)
	if err != nil {
		return err
	}
	cm.cluster.Spec.ClusterAPI.Status.APIEndpoints = append(cm.cluster.Spec.ClusterAPI.Status.APIEndpoints, clusterapi.APIEndpoint{
		Host: u.Host,
		Port: 0,
	})
	return nil
}

func (cm *ClusterManager) StoreCertificate(ctx context.Context, c *godo.Client, owner string) error {
	kcc, _, err := c.Kubernetes.GetKubeConfig(ctx, cm.cluster.Spec.Config.Cloud.Dokube.ClusterID)
	if err != nil {
		return err
	}

	kc, err := clientcmd.Load(kcc.KubeconfigYAML)
	if err != nil {
		return err
	}

	currentContext := kc.CurrentContext

	certStore := store.StoreProvider.Certificates(cm.cluster.Name)
	_, caKey, err := certStore.Get(api.CACertName)
	if err == nil {
		if err = certStore.Delete(api.CACertName); err != nil {
			return err
		}
	}

	caCrt, err := cert.ParseCertsPEM(kc.Clusters[currentContext].CertificateAuthorityData)
	if err != nil {
		return err
	}

	if err := certStore.Create(api.CACertName, caCrt[0], caKey); err != nil {
		return err
	}

	adminCrt, err := cert.ParseCertsPEM(kc.AuthInfos[kc.Contexts[currentContext].AuthInfo].ClientCertificateData)
	if err != nil {
		return err
	}

	adminKey, err := cert.ParsePrivateKeyPEM(kc.AuthInfos[kc.Contexts[currentContext].AuthInfo].ClientKeyData)
	if err != nil {
		return err
	}
	err = certStore.Create("admin", adminCrt[0], adminKey.(*rsa.PrivateKey))
	return err
}
