package v1beta1

import (
	"fmt"

	version "github.com/appscode/go-version"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	clusterapi "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	ResourceCodeCluster = ""
	ResourceKindCluster = "Cluster"
	ResourceNameCluster = "cluster"
	ResourceTypeCluster = "clusters"

	DefaultKubernetesBindPort = 6443
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PharmerClusterSpec   `json:"spec,omitempty"`
	Status            PharmerClusterStatus `json:"status,omitempty"`
}

type PharmerClusterSpec struct {
	ClusterAPI *clusterapi.Cluster `json:"clusterApi,omitempty"`
	Config     *ClusterConfig      `json:"config,omitempty"`
}

type ClusterConfig struct {
	MasterCount          int       `json:"masterCount"`
	Cloud                CloudSpec `json:"cloud"`
	KubernetesVersion    string    `json:"kubernetesVersion,omitempty"`
	Locked               bool      `json:"locked,omitempty"`
	CACertName           string    `json:"caCertName,omitempty"`
	FrontProxyCACertName string    `json:"frontProxyCACertName,omitempty"`
	CredentialName       string    `json:"credentialName,omitempty"`

	KubeletExtraArgs           map[string]string `json:"kubeletExtraArgs,omitempty"`
	APIServerExtraArgs         map[string]string `json:"apiServerExtraArgs,omitempty"`
	ControllerManagerExtraArgs map[string]string `json:"controllerManagerExtraArgs,omitempty"`
	SchedulerExtraArgs         map[string]string `json:"schedulerExtraArgs,omitempty"`
	AuthorizationModes         []string          `json:"authorizationModes,omitempty"`
	APIServerCertSANs          []string          `json:"apiServerCertSANs,omitempty"`
}

type API struct {
	// AdvertiseAddress sets the address for the API server to advertise.
	AdvertiseAddress string `json:"advertiseAddress"`
	// BindPort sets the secure port for the API Server to bind to
	BindPort int32 `json:"bindPort"`
}

type CloudSpec struct {
	CloudProvider        string      `json:"cloudProvider,omitempty"`
	Project              string      `json:"project,omitempty"`
	Region               string      `json:"region,omitempty"`
	Zone                 string      `json:"zone,omitempty"` // master needs it for ossec
	InstanceImage        string      `json:"instanceImage,omitempty"`
	OS                   string      `json:"os,omitempty"`
	InstanceImageProject string      `json:"instanceImageProject,omitempty"`
	NetworkProvider      string      `json:"networkProvider,omitempty"` // kubenet, flannel, calico, opencontrail
	CCMCredentialName    string      `json:"ccmCredentialName,omitempty"`
	SSHKeyName           string      `json:"sshKeyName,omitempty"`
	AWS                  *AWSSpec    `json:"aws,omitempty"`
	GCE                  *GoogleSpec `json:"gce,omitempty"`
	Azure                *AzureSpec  `json:"azure,omitempty"`
	Linode               *LinodeSpec `json:"linode,omitempty"`
	GKE                  *GKESpec    `json:"gke,omitempty"`
	//DigitalOcean         *DigitalOceanMachineProviderConfig `json:"digitalocean,omitempty"`
	Dokube *DokubeSpec `json:"dokube,omitempty"`
}

type AWSSpec struct {
	// aws:TAG KubernetesCluster => clusterid
	IAMProfileMaster  string `json:"iamProfileMaster,omitempty"`
	IAMProfileNode    string `json:"iamProfileNode,omitempty"`
	MasterSGName      string `json:"masterSGName,omitempty"`
	NodeSGName        string `json:"nodeSGName,omitempty"`
	BastionSGName     string `json:"bastionSGName,omitempty"`
	VpcCIDR           string `json:"vpcCIDR,omitempty"`
	VpcCIDRBase       string `json:"vpcCIDRBase,omitempty"`
	MasterIPSuffix    string `json:"masterIPSuffix,omitempty"`
	PrivateSubnetCIDR string `json:"privateSubnetCidr,omitempty"`
	PublicSubnetCIDR  string `json:"publicSubnetCidr,omitempty"`
}

type GoogleSpec struct {
	NetworkName string   `gcfg:"network-name" ini:"network-name,omitempty"`
	NodeTags    []string `gcfg:"node-tags" ini:"node-tags,omitempty"`
	// gce
	// NODE_SCOPES="${NODE_SCOPES:-compute-rw,monitoring,logging-write,storage-ro}"
	NodeScopes []string `json:"nodeScopes,omitempty"`
}

type GCECloudConfig struct {
	TokenURL           string   `gcfg:"token-url" ini:"token-url,omitempty"`
	TokenBody          string   `gcfg:"token-body" ini:"token-body,omitempty"`
	ProjectID          string   `gcfg:"project-id" ini:"project-id,omitempty"`
	NetworkName        string   `gcfg:"network-name" ini:"network-name,omitempty"`
	NodeTags           []string `gcfg:"node-tags" ini:"node-tags,omitempty"`
	NodeInstancePrefix string   `gcfg:"node-instance-prefix" ini:"node-instance-prefix,omitempty"`
	Multizone          bool     `gcfg:"multizone" ini:"multizone,omitempty"`
}

type GKESpec struct {
	UserName    string `json:"userName,omitempty"`
	Password    string `json:"password,omitempty"`
	NetworkName string `json:"networkName,omitempty"`
}

type AzureSpec struct {
	InstanceImageVersion   string `json:"instanceImageVersion,omitempty"`
	RootPassword           string `json:"rootPassword,omitempty"`
	VPCCIDR                string `json:"vpcCIDR"`
	ControlPlaneSubnetCIDR string `json:"controlPlaneSubnetCIDR"`
	NodeSubnetCIDR         string `json:"nodeSubnetCIDR"`
	InternalLBIPAddress    string `json:"internalLBIPAddress"`
	AzureDNSZone           string `json:"azureDNSZone"`
	SubnetCIDR             string `json:"subnetCidr,omitempty"`
	ResourceGroup          string `json:"resourceGroup,omitempty"`
	SubnetName             string `json:"subnetName,omitempty"`
	SecurityGroupName      string `json:"securityGroupName,omitempty"`
	VnetName               string `json:"vnetName,omitempty"`
	RouteTableName         string `json:"routeTableName,omitempty"`
	StorageAccountName     string `json:"azureStorageAccountName,omitempty"`
	SubscriptionID         string `json:"subscriptionID"`
}

// ref: https://github.com/kubernetes/kubernetes/blob/8b9f0ea5de2083589f3b9b289b90273556bc09c4/pkg/cloudprovider/providers/azure/azure.go#L56
type AzureCloudConfig struct {
	Cloud                        string  `json:"cloud"`
	TenantID                     string  `json:"tenantId,omitempty"`
	SubscriptionID               string  `json:"subscriptionId,omitempty"`
	AadClientID                  string  `json:"aadClientId,omitempty"`
	AadClientSecret              string  `json:"aadClientSecret,omitempty"`
	ResourceGroup                string  `json:"resourceGroup,omitempty"`
	Location                     string  `json:"location,omitempty"`
	VMType                       string  `json:"vmType"`
	SubnetName                   string  `json:"subnetName,omitempty"`
	SecurityGroupName            string  `json:"securityGroupName,omitempty"`
	VnetName                     string  `json:"vnetName,omitempty"`
	RouteTableName               string  `json:"routeTableName,omitempty"`
	PrimaryAvailabilitySetName   string  `json:"primaryAvailabilitySetName"`
	PrimaryScaleSetName          string  `json:"primaryScaleSetName"`
	CloudProviderBackoff         bool    `json:"cloudProviderBackoff"`
	CloudProviderBackoffRetries  int     `json:"cloudProviderBackoffRetries"`
	CloudProviderBackoffExponent float32 `json:"cloudProviderBackoffExponent"`
	CloudProviderBackoffDuration int     `json:"cloudProviderBackoffDuration"`
	CloudProviderBackoffJitter   float32 `json:"cloudProviderBackoffJitter"`
	CloudProviderRatelimit       bool    `json:"cloudProviderRatelimit"`
	CloudProviderRateLimitQPS    float32 `json:"cloudProviderRateLimitQPS"`
	CloudProviderRateLimitBucket int     `json:"cloudProviderRateLimitBucket"`
	UseManagedIdentityExtension  bool    `json:"useManagedIdentityExtension"`
	UserAssignedIdentityID       string  `json:"userAssignedIdentityID"`
	UseInstanceMetadata          bool    `json:"useInstanceMetadata"`
	LoadBalancerSku              string  `json:"loadBalancerSku"`
	ExcludeMasterFromStandardLB  bool    `json:"excludeMasterFromStandardLB"`
	ProviderVaultName            string  `json:"providerVaultName"`
	MaximumLoadBalancerRuleCount int     `json:"maximumLoadBalancerRuleCount"`
	ProviderKeyName              string  `json:"providerKeyName"`
	ProviderKeyVersion           string  `json:"providerKeyVersion"`
}

type LinodeSpec struct {
	// Linode
	RootPassword string `json:"rootPassword,omitempty"`
	KernelId     string `json:"kernelId,omitempty"`
}

type LinodeCloudConfig struct {
	Token string `json:"token,omitempty"`
	Zone  string `json:"zone,omitempty"`
}

type PacketCloudConfig struct {
	Project string `json:"project,omitempty"`
	ApiKey  string `json:"apiKey,omitempty"`
	Zone    string `json:"zone,omitempty"`
}

type VultrCloudConfig struct {
	Token string `json:"token,omitempty"`
}

type DokubeSpec struct {
	ClusterID string `json:"clusterID,omitempty"`
}

// ClusterPhase is a label for the condition of a Cluster at the current time.
type ClusterPhase string

// These are the valid statuses of Cluster.
const (
	ClusterPending   ClusterPhase = "Pending"
	ClusterReady     ClusterPhase = "Ready"
	ClusterDeleting  ClusterPhase = "Deleting"
	ClusterDeleted   ClusterPhase = "Deleted"
	ClusterUpgrading ClusterPhase = "Upgrading"
)

type CloudStatus struct {
	SShKeyExternalID string       `json:"sshKeyExternalID,omitempty"`
	AWS              *AWSStatus   `json:"aws,omitempty"`
	EKS              *EKSStatus   `json:"eks,omitempty"`
	LoadBalancer     LoadBalancer `json:"loadBalancer,omitempty"`
}

type LoadBalancer struct {
	DNS  string `json:"dns"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type AWSStatus struct {
	MasterSGId  string `json:"masterSGID,omitempty"`
	NodeSGId    string `json:"nodeSGID,omitempty"`
	BastionSGId string `json:"bastionSGID,omitempty"`
}

type EKSStatus struct {
	SecurityGroup string `json:"securityGroup,omitempty"`
	VpcId         string `json:"vpcID,omitempty"`
	SubnetId      string `json:"subnetID,omitempty"`
	RoleArn       string `json:"roleArn,omitempty"`
}

type PharmerClusterStatus struct {
	Phase  ClusterPhase `json:"phase,omitempty"`
	Reason string       `json:"reason,omitempty"`
	Cloud  CloudStatus  `json:"cloud,omitempty"`
	//ReservedIPs  []ReservedIP       `json:"reservedIP,omitempty" protobuf:"bytes,6,rep,name=reservedIP"`
}

type ReservedIP struct {
	IP   string `json:"ip,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func (c *Cluster) ClusterConfig() *ClusterConfig {
	return c.Spec.Config
}

func (c *Cluster) APIServerURL() string {
	for _, addr := range c.Spec.ClusterAPI.Status.APIEndpoints {
		if addr.Port == 0 {
			return fmt.Sprintf("https://%s", addr.Host)
		} else {
			return fmt.Sprintf("https://%s:%d", addr.Host, addr.Port)
		}

	}
	return ""
}

func (c *Cluster) SetClusterApiEndpoints(addresses []core.NodeAddress) error {
	m := map[core.NodeAddressType]string{}
	for _, addr := range addresses {
		m[addr.Type] = addr.Address

	}
	if u, found := m[core.NodeExternalIP]; found {
		c.Spec.ClusterAPI.Status.APIEndpoints = append(c.Spec.ClusterAPI.Status.APIEndpoints, clusterapi.APIEndpoint{
			Host: u,
			Port: int(DefaultKubernetesBindPort),
		})
		return nil
	}
	if u, found := m[core.NodeExternalDNS]; found {
		c.Spec.ClusterAPI.Status.APIEndpoints = append(c.Spec.ClusterAPI.Status.APIEndpoints, clusterapi.APIEndpoint{
			Host: u,
			Port: int(DefaultKubernetesBindPort),
		})
		return nil
	}
	return fmt.Errorf("No cluster api endpoint found")
}

func (c *Cluster) APIServerAddress() string {
	endpoints := c.Spec.ClusterAPI.Status.APIEndpoints
	if len(endpoints) == 0 {
		return ""
	}
	ep := endpoints[0]
	if ep.Port == 0 {
		return ep.Host
	} else {
		return fmt.Sprintf("%s:%d", ep.Host, ep.Port)
	}

}

func (c *Cluster) SetNetworkingDefaults(provider string) {
	clusterSpec := &c.Spec.ClusterAPI.Spec
	if len(clusterSpec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		clusterSpec.ClusterNetwork.Services.CIDRBlocks = []string{kubeadmapi.DefaultServicesSubnet}
	}
	if clusterSpec.ClusterNetwork.ServiceDomain == "" {
		clusterSpec.ClusterNetwork.ServiceDomain = kubeadmapi.DefaultServiceDNSDomain
	}
	if len(clusterSpec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#pod-network
		podSubnet := ""
		switch provider {
		case PodNetworkCalico:
			podSubnet = "192.168.0.0/16"
		case PodNetworkFlannel:
			podSubnet = "10.244.0.0/16"
		case PodNetworkCanal:
			podSubnet = "10.244.0.0/16"
		}
		clusterSpec.ClusterNetwork.Pods.CIDRBlocks = []string{podSubnet}
	}
}

func (c *Cluster) InitClusterApi() {
	c.Spec.ClusterAPI = &clusterapi.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Name,
		},
	}
}

func (c Cluster) IsMinorVersion(in string) bool {
	v, err := version.NewVersion(c.Spec.Config.KubernetesVersion)
	if err != nil {
		return false
	}
	minor := v.ToMutator().ResetMetadata().ResetPrerelease().ResetPatch().String()

	inVer, err := version.NewVersion(in)
	if err != nil {
		return false
	}
	return inVer.String() == minor
}

func (c Cluster) IsLessThanVersion(in string) bool {
	v, err := version.NewVersion(c.Spec.Config.KubernetesVersion)
	if err != nil {
		return false
	}
	inVer, err := version.NewVersion(in)
	if err != nil {
		return false
	}
	return v.LessThan(inVer)
}
