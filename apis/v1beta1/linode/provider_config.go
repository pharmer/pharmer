package linode_config

import (
	"encoding/json"

	. "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	LinodeProviderGroupName  = "linodeproviderconfig"
	LinodeProviderKind       = "LinodeClusterProviderConfig"
	LinodeProviderApiVersion = "v1alpha1"
)

type LinodeMachineProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Roles  []MachineRole `json:"roles,omitempty"`
	Region string        `json:"region,omitempty"`
	Type   string        `json:"type,omitempty"`
	Image  string        `json:"image,omitempty"`
	Pubkey string        `json:"pubkey,omitempty"`
}

type LinodeClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

//func (c *Cluster) SetLinodeProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
func SetLinodeClusterProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	conf := &LinodeMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: LinodeProviderGroupName + "/" + LinodeProviderApiVersion,
			Kind:       LinodeProviderKind,
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

func SetLinodeClusterProviderStatus(cluster *clusterapi.Cluster) error {
	conf := &LinodeClusterProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: LinodeProviderGroupName + "/" + LinodeProviderApiVersion,
			Kind:       LinodeProviderKind,
		},
	}
	bytes, err := json.Marshal(conf)
	if err != nil {
		return err

	}
	cluster.Status.ProviderStatus = &runtime.RawExtension{
		Raw: bytes,
	}
	return nil
}
