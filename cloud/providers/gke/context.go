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
package gke

import (
	"fmt"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cloud/utils/certificates"

	"github.com/pkg/errors"
	"gomodules.xyz/cert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	defaultNetwork = "default"
)

type ClusterManager struct {
	*cloud.Scope

	namer namer
	conn  *cloudConnector
}

const (
	UID = "gke"
)

func init() {
	cloud.RegisterCloudManager(UID, New)
}

func New(s *cloud.Scope) cloud.Interface {
	return &ClusterManager{
		Scope: s,
	}
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	return nil
}

func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	return nil
}

func (cm *ClusterManager) SetCloudConnector() error {
	if cm.conn != nil {
		return nil
	}
	conn, err := newconnector(cm)
	cm.conn = conn
	return err
}

func (cm *ClusterManager) NewMasterTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	return cloud.TemplateData{}
}

func (cm *ClusterManager) NewNodeTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	return cloud.TemplateData{}
}

func (cm *ClusterManager) EnsureMaster(_ *v1alpha1.Machine) error {
	return nil
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	return ""
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return "", nil
}

var _ cloud.Interface = &ClusterManager{}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	kc, err := cm.NewGKEAdminClient()
	if err != nil {
		return nil, err
	}
	return kc, nil
}

func (cm *ClusterManager) NewGKEAdminClient() (kubernetes.Interface, error) {
	log := cm.Logger

	cluster := cm.Cluster
	adminCert, adminKey, err := certificates.GetAdminCertificate(cm.StoreProvider.Certificates(cm.Cluster.Name))
	if err != nil {
		log.Error(err, "failed to get admin certificates")
		return nil, err
	}
	host := cluster.APIServerURL()
	if host == "" {
		err = errors.Errorf("failed to detect api server url for cluster %s", cluster.Name)
		log.Error(err, "apiserver url is empty")
		return nil, err
	}
	cfg := &rest.Config{
		Host:     host,
		Username: cluster.Spec.Config.Cloud.GKE.UserName,
		Password: cluster.Spec.Config.Cloud.GKE.Password,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cert.EncodeCertPEM(cm.GetCaCertPair().Cert),
			CertData: cert.EncodeCertPEM(adminCert),
			KeyData:  cert.EncodePrivateKeyPEM(adminKey),
		},
	}

	return kubernetes.NewForConfig(cfg)
}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	cluster := cm.Cluster

	var (
		clusterName = fmt.Sprintf("%s.pharmer", cluster.Name)
		userName    = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
		ctxName     = fmt.Sprintf("cluster-admin@%s.pharmer", cluster.Name)
	)
	cfg := api.KubeConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "KubeConfig",
		},
		Preferences: api.Preferences{
			Colors: true,
		},
		Cluster: api.NamedCluster{
			Name:                     clusterName,
			Server:                   cluster.APIServerURL(),
			CertificateAuthorityData: cert.EncodeCertPEM(cm.GetCaCertPair().Cert),
		},
		AuthInfo: api.NamedAuthInfo{
			Name:     userName,
			Username: cluster.Spec.Config.Cloud.GKE.UserName,
			Password: cluster.Spec.Config.Cloud.GKE.Password,
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}
