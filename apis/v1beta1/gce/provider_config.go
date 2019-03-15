package gce

import (
	"encoding/json"
	"fmt"

	. "github.com/pharmer/pharmer/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	GCEProviderGroupName   = "gceproviderconfig"
	GCEClusterProviderKind = "GCEClusterProviderSpec"
	GCEMachineProviderKind = "GCEMachineProviderSpec"
	GCEProviderApiVersion  = "v1alpha1"
)

// GCEMachineProviderSpec is the Schema for the gcemachineproviderconfigs API
// +k8s:openapi-gen=true
type GCEMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Roles             []MachineRole `json:"roles,omitempty"`

	Zone        string `json:"zone"`
	MachineType string `json:"machineType"`

	// The name of the OS to be installed on the machine.
	OS    string `json:"os,omitempty"`
	Disks []Disk `json:"disks,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GCEClusterProviderSpec is the Schema for the gceclusterproviderconfigs API
// +k8s:openapi-gen=true
type GCEClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Project string `json:"project"`
}

type GCEClusteProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

//func (c *cluster) GCEProviderSpec(cluster *clusterapi.Cluster) *GCEMachineProviderSpec {
func GetGCEMachineProviderSpec(providerSPec clusterapi.ProviderSpec) *GCEMachineProviderSpec {
	raw := providerSPec.Value.Raw
	providerConfig := &GCEMachineProviderSpec{}
	err := json.Unmarshal(raw, providerConfig)
	if err != nil {
		fmt.Println("Unable to unmarshal provider config: %v", err)
	}
	return providerConfig
}

//func (c *Cluster) SetGCEProviderSpec( cluster *clusterapi.Cluster, config *ClusterConfig) error {
func SetGCEClusterProviderSpec(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	fmt.Println(config.Cloud)
	conf := &GCEClusterProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GCEProviderGroupName + "/" + GCEProviderApiVersion,
			Kind:       GCEClusterProviderKind,
		},
		Project: config.Cloud.Project,
	}
	bytes, err := json.Marshal(conf)
	if err != nil {
		fmt.Println("Unable to marshal provider config: %v", err)
		return err
	}
	cluster.Spec.ProviderSpec = clusterapi.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: bytes,
		},
	}
	return nil
}

func SetGCEClusterProviderStatus(cluster *clusterapi.Cluster) error {
	conf := &GCEClusteProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GCEProviderGroupName + "/" + GCEProviderApiVersion,
			Kind:       GCEClusterProviderKind,
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
