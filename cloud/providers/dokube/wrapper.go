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
	"crypto/rsa"
	"net/url"

	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/digitalocean/godo"
	"gomodules.xyz/cert"
	"k8s.io/client-go/tools/clientcmd"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func (cm *ClusterManager) retrieveClusterStatus(cluster *godo.KubernetesCluster) error {
	log := cm.Logger
	u, err := url.Parse(cluster.Endpoint)
	if err != nil {
		log.Error(err, "failed to parse url", "endpoint", cluster.Endpoint)
		return err
	}
	cm.Cluster.Spec.ClusterAPI.Status.APIEndpoints = append(cm.Cluster.Spec.ClusterAPI.Status.APIEndpoints, clusterapi.APIEndpoint{
		Host: u.Host,
		Port: 0,
	})
	return nil
}

func (cm *ClusterManager) StoreCertificate(c *godo.Client) error {
	log := cm.Logger
	kcc, _, err := c.Kubernetes.GetKubeConfig(context.Background(), cm.Cluster.Spec.Config.Cloud.Dokube.ClusterID)
	if err != nil {
		log.Error(err, "failed to get kubeconfig from digitalocean cluster")
		return err
	}

	kc, err := clientcmd.Load(kcc.KubeconfigYAML)
	if err != nil {
		log.Error(err, "failed to load kubeconfig")
		return err
	}

	currentContext := kc.CurrentContext

	certStore := cm.StoreProvider.Certificates(cm.Cluster.Name)
	_, caKey, err := certStore.Get(api.CACertName)
	if err == nil {
		if err = certStore.Delete(api.CACertName); err != nil {
			return err
		}
	}

	caCrt, err := cert.ParseCertsPEM(kc.Clusters[currentContext].CertificateAuthorityData)
	if err != nil {
		log.Error(err, "failed to parse ca-cert pem")
		return err
	}

	if err := certStore.Create(api.CACertName, caCrt[0], caKey); err != nil {
		log.Error(err, "failed to create ca-cert in store")
		return err
	}

	adminCrt, err := cert.ParseCertsPEM(kc.AuthInfos[kc.Contexts[currentContext].AuthInfo].ClientCertificateData)
	if err != nil {
		log.Error(err, "failed to parse admin certs")
		return err
	}

	adminKey, err := cert.ParsePrivateKeyPEM(kc.AuthInfos[kc.Contexts[currentContext].AuthInfo].ClientKeyData)
	if err != nil {
		log.Error(err, "failed to parse admin key")
		return err
	}
	err = certStore.Create("admin", adminCrt[0], adminKey.(*rsa.PrivateKey))
	if err != nil {
		log.Error(err, "failed to create admin certs & key in store")
		return err
	}
	return nil
}
