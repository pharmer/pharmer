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
package gce

import (
	"pharmer.dev/pharmer/cloud"

	"k8s.io/client-go/kubernetes"
)

const (
	defaultNetwork = "default"
)

type ClusterManager struct {
	*cloud.Scope

	conn  *cloudConnector
	namer namer
}

func (cm *ClusterManager) ApplyScale() error {
	panic("implement me")
}

var _ cloud.Interface = &ClusterManager{}

const (
	UID = "gce"
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

// TODO: Verify
func (cm *ClusterManager) CreateCredentials(kc kubernetes.Interface) error {
	//cloud.CreateCredentialSecret(cm.AdminClient, cm.Cluster)
	return nil
}
