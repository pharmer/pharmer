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
	"encoding/json"

	api "pharmer.dev/pharmer/apis/v1alpha1"

	"github.com/appscode/go/encoding/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	DokubeProviderGroupName  = "dokubeproviderconfig"
	DokubeProviderKind       = "DokubeClusterProviderConfig"
	DokubeProviderAPIVersion = "v1alpha1"
)

type DokubeMachineProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Size string `json:"size,omitempty"`
}

func SetLDokubeClusterProviderConfig(cluster *clusterapi.Cluster, config *api.ClusterConfig) error {
	conf := &DokubeMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: DokubeProviderGroupName + "/" + DokubeProviderAPIVersion,
			Kind:       DokubeProviderKind,
		},
	}
	bytes, err := json.Marshal(conf)
	if err != nil {
		return err

	}
	cluster.Spec.ProviderSpec = clusterapi.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: bytes,
		},
	}
	return nil
}

func DokubeProviderConfig(byteConfig []byte) (*DokubeMachineProviderConfig, error) {
	var config DokubeMachineProviderConfig
	if err := yaml.Unmarshal(byteConfig, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
