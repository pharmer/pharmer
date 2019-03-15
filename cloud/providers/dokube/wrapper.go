package dokube

import (
	"context"
	"crypto/rsa"
	"net/url"

	"github.com/digitalocean/godo"
	. "github.com/pharmer/pharmer/cloud"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
)

func (cm *ClusterManager) retrieveClusterStatus(cluster *godo.KubernetesCluster) error {
	u, err := url.Parse(cluster.Endpoint)
	if err != nil {
		return err
	}
	cm.cluster.Status.APIAddresses = append(cm.cluster.Status.APIAddresses, core.NodeAddress{
		Type:    core.NodeExternalIP,
		Address: u.Host,
	})
	return nil
}

func (cm *ClusterManager) StoreCertificate(ctx context.Context, c *godo.Client) error {
	kcc, _, err := c.Kubernetes.GetKubeConfig(ctx, cm.cluster.Spec.Cloud.Dokube.ClusterID)
	if err != nil {
		return err
	}

	kc, err := clientcmd.Load(kcc.KubeconfigYAML)
	if err != nil {
		return err
	}

	currentContext := kc.CurrentContext

	certStore := Store(cm.ctx).Certificates(cm.cluster.Name)
	_, caKey, err := certStore.Get(cm.cluster.Spec.CACertName)
	if err == nil {
		if err = certStore.Delete(cm.cluster.Spec.CACertName); err != nil {
			return err
		}
	}

	caCrt, err := cert.ParseCertsPEM(kc.Clusters[currentContext].CertificateAuthorityData)
	if err != nil {
		return err
	}

	if err := certStore.Create(cm.cluster.Spec.CACertName, caCrt[0], caKey); err != nil {
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
