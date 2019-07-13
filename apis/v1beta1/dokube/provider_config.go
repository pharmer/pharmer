package dokube

import (
	"encoding/json"

	"github.com/appscode/go/encoding/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	api "pharmer.dev/pharmer/apis/v1beta1"
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
