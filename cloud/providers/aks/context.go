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
package aks

import (
	"context"
	"encoding/json"
	"fmt"

	api "pharmer.dev/pharmer/apis/v1alpha1"
	"pharmer.dev/pharmer/cloud"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterManager struct {
	*cloud.Scope

	conn *cloudConnector

	namer namer
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID             = "aks"
	RoleClusterUser = "clusterUser"
)

func init() {
	cloud.RegisterCloudManager(UID, New)
}

func New(s *cloud.Scope) cloud.Interface {
	return &ClusterManager{
		Scope: s,
		namer: namer{cluster: s.Cluster},
	}
}

func (cm *ClusterManager) AddToManager(m manager.Manager) error {
	panic("implement me")
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	return nil
}

func (cm *ClusterManager) SetCloudConnector() error {
	conn, err := newconnector(cm)
	cm.conn = conn
	return err
}

func (cm *ClusterManager) NewMasterTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	panic("implement me")
}

func (cm *ClusterManager) NewNodeTemplateData(machine *v1alpha1.Machine, token string, td cloud.TemplateData) cloud.TemplateData {
	panic("implement me")
}

func (cm *ClusterManager) EnsureMaster(_ *v1alpha1.Machine) error {
	return nil
}

func (cm *ClusterManager) GetMasterSKU(totalNodes int32) string {
	return ""
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	panic("implement me")
}

func (cm *ClusterManager) GetAdminClient() (kubernetes.Interface, error) {
	if cm.AdminClient != nil {
		return cm.AdminClient, nil
	}
	client, err := cm.GetAKSAdminClient()
	cm.AdminClient = client
	return cm.AdminClient, err
}

func (cm *ClusterManager) GetAKSAdminClient() (kubernetes.Interface, error) {
	log := cm.Logger
	resp, err := cm.conn.managedClient.GetAccessProfile(context.Background(), cm.namer.ResourceGroupName(), cm.Cluster.Name, RoleClusterUser)
	if err != nil {
		log.Error(err, "failed to get access profile")
		return nil, err
	}
	fmt.Println(*resp.KubeConfig)
	kubeconfig := *resp.KubeConfig
	kubeconfig, err = yaml.YAMLToJSON(kubeconfig)
	if err != nil {
		log.Error(err, "failed to convert kubeconfig from yaml to json")
		return nil, err
	}
	var konfig clientcmd.Config
	err = json.Unmarshal(kubeconfig, &konfig)
	if err != nil {
		log.Error(err, "failed to unmarshal kubeconfig")
		return nil, err
	}

	cfg := &rest.Config{
		Host: konfig.Clusters[0].Cluster.Server,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   konfig.Clusters[0].Cluster.CertificateAuthorityData,
			CertData: konfig.AuthInfos[0].AuthInfo.ClientCertificateData,
			KeyData:  konfig.AuthInfos[0].AuthInfo.ClientKeyData,
		},
	}
	return kubernetes.NewForConfig(cfg)

}

func (cm *ClusterManager) GetKubeConfig() (*api.KubeConfig, error) {
	var err error
	log := cm.Logger
	cluster := cm.Cluster
	cm.namer = namer{cluster: cluster}
	if cm.conn, err = newconnector(cm); err != nil {
		log.Error(err, "failed to get cloud connector")
		return nil, err
	}

	resp, err := cm.conn.managedClient.GetAccessProfile(context.Background(), cm.namer.ResourceGroupName(), cm.Cluster.Name, RoleClusterUser)
	if err != nil {
		log.Error(err, "failed to get access profile")
		return nil, err
	}

	kubeconfig := *resp.KubeConfig
	kubeconfig, err = yaml.YAMLToJSON(kubeconfig)
	if err != nil {
		log.Error(err, "failed to convert kubeconfig from yaml to json")
		return nil, err
	}

	var konfig clientcmd.Config
	err = json.Unmarshal(kubeconfig, &konfig)
	if err != nil {
		log.Error(err, "failed to unmarshal kubeconfig")
		return nil, err
	}
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
			Server:                   konfig.Clusters[0].Cluster.Server,
			CertificateAuthorityData: konfig.Clusters[0].Cluster.CertificateAuthorityData,
		},
		AuthInfo: api.NamedAuthInfo{
			Name:                  userName,
			ClientCertificateData: konfig.AuthInfos[0].AuthInfo.ClientCertificateData,
			ClientKeyData:         konfig.AuthInfos[0].AuthInfo.ClientKeyData,
		},
		Context: api.NamedContext{
			Name:     ctxName,
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	return &cfg, nil
}
