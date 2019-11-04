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
package linode

import (
	"pharmer.dev/pharmer/cloud"
	"pharmer.dev/pharmer/cloud/utils/kube"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ClusterManager struct {
	*cloud.Scope

	conn  *cloudConnector
	namer namer
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID      = "linode"
	Recorder = "linode-controller"
)

func init() {
	cloud.RegisterCloudManager(UID, New)
}

func New(s *cloud.Scope) cloud.Interface {
	return &ClusterManager{
		Scope: s,
		namer: namer{
			cluster: s.Cluster,
		},
	}
}

func (cm *ClusterManager) ApplyScale() error {
	panic("implement me")
}

func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	log := cm.Logger
	cred, err := cm.GetCredential()
	if err != nil {
		log.Error(err, "failed to get credential from store")
		return err
	}
	err = kube.CreateCredentialSecret(kc, cm.Cluster.CloudProvider(), metav1.NamespaceSystem, cred.Spec.Data)
	if err != nil {
		log.Error(err, "failed to create credential for pharmer-flex")
		return err
	}

	err = kube.CreateSecret(kc, "ccm-linode", metav1.NamespaceSystem, map[string][]byte{
		"apiToken": []byte(cred.Spec.Data["token"]),
		"region":   []byte(cm.Cluster.ClusterConfig().Cloud.Region),
	})
	if err != nil {
		log.Error(err, "failed to create ccm-secret")
		return err
	}
	return nil
}

func (cm *ClusterManager) SetCloudConnector() error {
	var err error

	if cm.conn, err = newconnector(cm); err != nil {
		cm.Logger.Error(err, "failed to get linode cloud connector")
		return err
	}

	return nil
}

func (cm *ClusterManager) GetClusterAPIComponents() (string, error) {
	return ControllerManager, nil
}
