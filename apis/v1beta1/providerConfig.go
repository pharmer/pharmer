package v1beta1

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	DigitalOceanProviderGroupName  = "digitaloceanproviderconfig"
	DigitalOceanProviderKind       = "DigitaloceanClusterProviderConfig"
	DigitalOceanProviderApiVersion = "v1alpha1"

	GKEProviderGroupName  = "gkeproviderconfig"
	GKEProviderKind       = "GKEProviderConfig"
	GKEProviderApiVersion = "v1alpha1"

	EKSProviderGroupName  = "eksproviderconfig"
	EKSProviderKind       = "EKSProviderConfig"
	EKSProviderApiVersion = "v1alpha1"

	AWSProviderGroupName   = "awsprovider"
	AWSProviderApiVersion  = "v1alpha1"
	AWSClusterProviderKind = "AWSClusterProviderSpec"
	AWSMachineProviderKind = "AWSMachineProviderSpec"

	AKSProviderGroupName  = "azureprovider"
	AKSProviderKind       = "AzureClusterProviderSpec"
	AKSProviderApiVersion = "v1alpha1"
)

// DigitalOceanMachineProviderConfig contains Config for DigitalOcean machines.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DigitalOceanMachineProviderConfig struct {
	metav1.TypeMeta `json:",inline"`

	Region        string   `json:"region,omitempty"`
	Size          string   `json:"size,omitempty"`
	Image         string   `json:"image,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	SSHPublicKeys []string `json:"sshPublicKeys,omitempty"`

	PrivateNetworking bool `json:"private_networking,omitempty"`
	Backups           bool `json:"backups,omitempty"`
	IPv6              bool `json:"ipv6,omitempty"`
	Monitoring        bool `json:"monitoring,omitempty"`
}

func (c *Cluster) DigitalOceanProviderConfig(cluster *clusterapi.Cluster) *DigitalOceanMachineProviderConfig {
	//providerConfig providerConfig
	raw := cluster.Spec.ProviderSpec.Value.Raw
	providerConfig := &DigitalOceanMachineProviderConfig{}
	err := json.Unmarshal(raw, providerConfig)
	if err != nil {
		fmt.Println("Unable to unmarshal provider config: %v", err)
	}
	return providerConfig
}

func (c *Cluster) SetDigitalOceanProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	conf := &DigitalOceanMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: DigitalOceanProviderGroupName + "/" + DigitalOceanProviderApiVersion,
			Kind:       DigitalOceanProviderKind,
		},
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

type GKEMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Roles []MachineRole `json:"roles,omitempty"`

	Zone        string `json:"zone"`
	MachineType string `json:"machineType"`

	// The name of the OS to be installed on the machine.
	OS    string `json:"os,omitempty"`
	Disks []Disk `json:"disks,omitempty"`
}

type Disk struct {
	InitializeParams DiskInitializeParams `json:"initializeParams"`
}

type DiskInitializeParams struct {
	DiskSizeGb int64  `json:"diskSizeGb"`
	DiskType   string `json:"diskType"`
}

func (c *Cluster) GKEProviderConfig(raw []byte) *GKEMachineProviderSpec {
	//providerConfig providerConfig
	providerConfig := &GKEMachineProviderSpec{}
	err := json.Unmarshal(raw, providerConfig)
	if err != nil {
		fmt.Println("Unable to unmarshal provider config: %v", err)
	}
	return providerConfig
}

func (c *Cluster) SetGKEProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	conf := &GKEMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GKEProviderGroupName + "/" + GKEProviderApiVersion,
			Kind:       GKEProviderKind,
		},
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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSMachineProviderSpec is the type that will be embedded in a Machine.Spec.ProviderSpec field
// for an AWS instance. It is used by the AWS machine actuator to create a single machine instance,
// using the RunInstances call (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RunInstances.html)
// Required parameters such as region that are not specified by this configuration, will be defaulted
// by the actuator.
// +k8s:openapi-gen=true
type EKSMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// AMI is the reference to the AMI from which to create the machine instance.
	AMI AWSResourceReference `json:"ami,omitempty"`

	// InstanceType is the type of instance to create. Example: m4.xlarge
	InstanceType string `json:"instanceType,omitempty"`

	// AdditionalTags is the set of tags to add to an instance, in addition to the ones
	// added by default by the actuator. These tags are additive. The actuator will ensure
	// these tags are present, but will not remove any other tags that may exist on the
	// instance.
	// +optional
	AdditionalTags map[string]string `json:"additionalTags,omitempty"`

	// IAMInstanceProfile is a name of an IAM instance profile to assign to the instance
	// +optional
	IAMInstanceProfile string `json:"iamInstanceProfile,omitempty"`

	// PublicIP specifies whether the instance should get a public IP.
	// Precedence for this setting is as follows:
	// 1. This field if set
	// 2. Cluster/flavor setting
	// 3. Subnet default
	// +optional
	PublicIP *bool `json:"publicIP,omitempty"`

	// AdditionalSecurityGroups is an array of references to security groups that should be applied to the
	// instance. These security groups would be set in addition to any security groups defined
	// at the cluster level or in the actuator.
	// +optional
	AdditionalSecurityGroups []AWSResourceReference `json:"additionalSecurityGroups,omitempty"`

	// Subnet is a reference to the subnet to use for this instance. If not specified,
	// the cluster subnet will be used.
	// +optional
	Subnet *AWSResourceReference `json:"subnet,omitempty"`

	// KeyName is the name of the SSH key to install on the instance.
	// +optional
	KeyName string `json:"keyName,omitempty"`
}
type AWSResourceReference struct {
	// ID of resource
	// +optional
	ID *string `json:"id,omitempty"`

	// ARN of resource
	// +optional
	ARN *string `json:"arn,omitempty"`

	// Filters is a set of key/value pairs used to identify a resource
	// They are applied according to the rules defined by the AWS API:
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/Using_Filtering.html
	// +optional
	Filters []Filter `json:"filters,omitempty"`
}

type Filter struct {
	// Name of the filter. Filter names are case-sensitive.
	Name string `json:"name"`

	// Values includes one or more filter values. Filter values are case-sensitive.
	Values []string `json:"values"`
}

func (c *Cluster) EKSProviderConfig(raw []byte) *EKSMachineProviderSpec {
	providerConfig := &EKSMachineProviderSpec{}
	err := json.Unmarshal(raw, providerConfig)
	if err != nil {
		fmt.Println("Unable to unmarshal provider config: %v", err)
	}
	return providerConfig
}
func (c *Cluster) SetEKSProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	conf := &EKSMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: EKSProviderGroupName + "/" + EKSProviderApiVersion,
			Kind:       EKSProviderKind,
		},
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

// AKSClusterProviderSpec is the Schema for the azureclusterproviderspecs API
// +k8s:openapi-gen=true
type AKSClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ResourceGroup string `json:"resourceGroup"`
	Location      string `json:"location"`

	// CACertificate is a PEM encoded CA Certificate for the control plane nodes.
	CACertificate []byte `json:"caCertificate,omitempty"`

	// CAPrivateKey is a PEM encoded PKCS1 CA PrivateKey for the control plane nodes.
	CAPrivateKey []byte `json:"caKey,omitempty"`
}

// AKSMachineProviderSpec is the Schema for the azuremachineproviderspecs API
// +k8s:openapi-gen=true
type AKSMachineProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Roles         []MachineRole `json:"roles,omitempty"`
	Location      string        `json:"location"`
	VMSize        string        `json:"vmSize"`
	Image         Image         `json:"image"`
	OSDisk        OSDisk        `json:"osDisk"`
	SSHPublicKey  string        `json:"sshPublicKey"`
	SSHPrivateKey string        `json:"sshPrivateKey"`
}

type Image struct {
	Publisher string `json:"publisher"`
	Offer     string `json:"offer"`
	SKU       string `json:"sku"`
	Version   string `json:"version"`
}

type OSDisk struct {
	OSType      string      `json:"osType"`
	ManagedDisk ManagedDisk `json:"managedDisk"`
	DiskSizeGB  int         `json:"diskSizeGB"`
}

type ManagedDisk struct {
	StorageAccountType string `json:"storageAccountType"`
}

func (c *Cluster) SetAKSProviderConfig(cluster *clusterapi.Cluster, config *ClusterConfig) error {
	conf := &GKEMachineProviderSpec{
		TypeMeta: metav1.TypeMeta{
			APIVersion: AKSProviderGroupName + "/" + AKSProviderApiVersion,
			Kind:       AKSProviderKind,
		},
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

func (c *Cluster) AKSProviderConfig(raw []byte) *AKSMachineProviderSpec {
	providerConfig := &AKSMachineProviderSpec{}
	err := json.Unmarshal(raw, providerConfig)
	if err != nil {
		fmt.Println("Unable to unmarshal provider config: %v", err)
	}
	return providerConfig
}
