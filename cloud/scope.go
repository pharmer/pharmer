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
package cloud

import (
	cloudapi "pharmer.dev/cloud/pkg/apis/cloud/v1"
	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud/utils/certificates"
	"pharmer.dev/pharmer/cloud/utils/kube"
	"pharmer.dev/pharmer/store"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/klogr"
)

type CloudManagerInterface interface {
	GetCluster() *api.Cluster
	GetCaCertPair() *certificates.CertKeyPair
	GetCertificates() *certificates.Certificates
	GetCredential() (*cloudapi.Credential, error)

	GetAdminClient() (kubernetes.Interface, error)
	GetCloudManager() (Interface, error)
}

var _ CloudManagerInterface = &Scope{}

type Scope struct {
	Cluster        *api.Cluster
	Certs          *certificates.Certificates
	CredentialData map[string]string
	StoreProvider  store.ResourceInterface
	CloudManager   Interface
	AdminClient    kubernetes.Interface
	logr.Logger
}

type NewScopeParams struct {
	Cluster       *api.Cluster
	Certs         *certificates.Certificates
	StoreProvider store.ResourceInterface
	Logger        logr.Logger
}

func NewScope(params NewScopeParams) *Scope {
	if params.Logger == nil {
		params.Logger = klogr.New().WithValues("cluster-name", params.Cluster.Name)
	}
	return &Scope{
		Cluster:       params.Cluster,
		Certs:         params.Certs,
		StoreProvider: params.StoreProvider,
		Logger:        params.Logger,
	}
}

func (s *Scope) GetCredential() (*cloudapi.Credential, error) {
	return s.StoreProvider.Credentials().Get(s.Cluster.Spec.Config.CredentialName)
}

func (s *Scope) GetCluster() *api.Cluster {
	return s.Cluster
}

func (s *Scope) GetCloudManager() (Interface, error) {
	if s.CloudManager != nil {
		return s.CloudManager, nil
	}
	var err error
	s.CloudManager, err = GetCloudManager(s)
	return s.CloudManager, err
}

func (s *Scope) GetCaCertPair() *certificates.CertKeyPair {
	return &s.Certs.CACert
}

func (s *Scope) GetCertificates() *certificates.Certificates {
	return s.Certs
}

func (s *Scope) GetAdminClient() (kubernetes.Interface, error) {
	log := s.Logger
	if s.AdminClient != nil {
		return s.AdminClient, nil
	}

	clusterEndpoint := s.Cluster.APIServerURL()
	if clusterEndpoint == "" {
		return nil, errors.Errorf("failed to detect api server url for Cluster %s", s.Cluster.Name)
	}

	client, err := kube.NewAdminClient(s.StoreProvider.Certificates(s.Cluster.Name), clusterEndpoint)
	if err != nil {
		log.Error(err, "failed to create new kube-client")
		return nil, err
	}
	s.AdminClient = client

	return client, nil
}
