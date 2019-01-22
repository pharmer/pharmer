package v1beta1

import (
	"fmt"

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
	metav1.TypeMeta   `json:",inline,omitempty,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              PharmerClusterSpec   `json:"spec,omitempty,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            PharmerClusterStatus `json:"status,omitempty,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type PharmerClusterSpec struct {
	ClusterAPI *clusterapi.Cluster `json:"clusterApi,omitempty" protobuf:"bytes,1,opt,name=clusterApi"`
	Config     *ClusterConfig      `json:"config,omitempty" protobuf:"bytes,2,opt,name=config"`
}
type ClusterConfig struct {
	Cloud                CloudSpec `json:"cloud" protobuf:"bytes,1,opt,name=cloud"`
	KubernetesVersion    string    `json:"kubernetesVersion,omitempty" protobuf:"bytes,4,opt,name=kubernetesVersion"`
	Locked               bool      `json:"locked,omitempty" protobuf:"varint,5,opt,name=locked"`
	CACertName           string    `json:"caCertName,omitempty" protobuf:"bytes,6,opt,name=caCertName"`
	FrontProxyCACertName string    `json:"frontProxyCACertName,omitempty" protobuf:"bytes,7,opt,name=frontProxyCACertName"`
	CredentialName       string    `json:"credentialName,omitempty" protobuf:"bytes,8,opt,name=credentialName"`

	KubeletExtraArgs           map[string]string `json:"kubeletExtraArgs,omitempty" protobuf:"bytes,9,rep,name=kubeletExtraArgs"`
	APIServerExtraArgs         map[string]string `json:"apiServerExtraArgs,omitempty" protobuf:"bytes,10,rep,name=apiServerExtraArgs"`
	ControllerManagerExtraArgs map[string]string `json:"controllerManagerExtraArgs,omitempty" protobuf:"bytes,11,rep,name=controllerManagerExtraArgs"`
	SchedulerExtraArgs         map[string]string `json:"schedulerExtraArgs,omitempty" protobuf:"bytes,12,rep,name=schedulerExtraArgs"`
	AuthorizationModes         []string          `json:"authorizationModes,omitempty" protobuf:"bytes,13,rep,name=authorizationModes"`
	APIServerCertSANs          []string          `json:"apiServerCertSANs,omitempty" protobuf:"bytes,14,rep,name=apiServerCertSANs"`
}

type API struct {
	// AdvertiseAddress sets the address for the API server to advertise.
	AdvertiseAddress string `json:"advertiseAddress" protobuf:"bytes,1,opt,name=advertiseAddress"`
	// BindPort sets the secure port for the API Server to bind to
	BindPort int32 `json:"bindPort" protobuf:"varint,2,opt,name=bindPort"`
}

type CloudSpec struct {
	CloudProvider        string      `json:"cloudProvider,omitempty" protobuf:"bytes,1,opt,name=cloudProvider"`
	Project              string      `json:"project,omitempty" protobuf:"bytes,2,opt,name=project"`
	Region               string      `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
	Zone                 string      `json:"zone,omitempty" protobuf:"bytes,4,opt,name=zone"` // master needs it for ossec
	InstanceImage        string      `json:"instanceImage,omitempty" protobuf:"bytes,5,opt,name=instanceImage"`
	OS                   string      `json:"os,omitempty" protobuf:"bytes,6,opt,name=os"`
	InstanceImageProject string      `json:"instanceImageProject,omitempty" protobuf:"bytes,7,opt,name=instanceImageProject"`
	NetworkProvider      string      `json:"networkProvider,omitempty" protobuf:"bytes,8,opt,name=networkProvider"` // kubenet, flannel, calico, opencontrail
	CCMCredentialName    string      `json:"ccmCredentialName,omitempty" protobuf:"bytes,9,opt,name=ccmCredentialName"`
	SSHKeyName           string      `json:"sshKeyName,omitempty" protobuf:"bytes,10,opt,name=sshKeyName"`
	AWS                  *AWSSpec    `json:"aws,omitempty" protobuf:"bytes,11,opt,name=aws"`
	GCE                  *GoogleSpec `json:"gce,omitempty" protobuf:"bytes,12,opt,name=gce"`
	Azure                *AzureSpec  `json:"azure,omitempty" protobuf:"bytes,13,opt,name=azure"`
	Linode               *LinodeSpec `json:"linode,omitempty" protobuf:"bytes,14,opt,name=linode"`
	GKE                  *GKESpec    `json:"gke,omitempty" protobuf:"bytes,15,opt,name=gke"`
	//DigitalOcean         *DigitalOceanMachineProviderConfig `json:"digitalocean,omitempty" protobuf:"bytes,16,opt,name=digitalocean"`
}

type AWSSpec struct {
	// aws:TAG KubernetesCluster => clusterid
	IAMProfileMaster string `json:"iamProfileMaster,omitempty" protobuf:"bytes,1,opt,name=iamProfileMaster"`
	IAMProfileNode   string `json:"iamProfileNode,omitempty" protobuf:"bytes,2,opt,name=iamProfileNode"`
	MasterSGName     string `json:"masterSGName,omitempty" protobuf:"bytes,3,opt,name=masterSGName"`
	NodeSGName       string `json:"nodeSGName,omitempty" protobuf:"bytes,4,opt,name=nodeSGName"`
	VpcCIDR          string `json:"vpcCIDR,omitempty" protobuf:"bytes,5,opt,name=vpcCIDR"`
	VpcCIDRBase      string `json:"vpcCIDRBase,omitempty" protobuf:"bytes,6,opt,name=vpcCIDRBase"`
	MasterIPSuffix   string `json:"masterIPSuffix,omitempty" protobuf:"bytes,7,opt,name=masterIPSuffix"`
	SubnetCIDR       string `json:"subnetCidr,omitempty" protobuf:"bytes,8,opt,name=subnetCidr"`
}

type GoogleSpec struct {
	NetworkName string   `gcfg:"network-name" ini:"network-name,omitempty" protobuf:"bytes,1,opt,name=networkName"`
	NodeTags    []string `gcfg:"node-tags" ini:"node-tags,omitempty,omitempty" protobuf:"bytes,2,rep,name=nodeTags"`
	// gce
	// NODE_SCOPES="${NODE_SCOPES:-compute-rw,monitoring,logging-write,storage-ro}"
	NodeScopes []string `json:"nodeScopes,omitempty" protobuf:"bytes,3,rep,name=nodeScopes"`
}

type GKESpec struct {
	UserName    string `json:"userName,omitempty" protobuf:"bytes,1,opt,name=userName"`
	Password    string `json:"password,omitempty" protobuf:"bytes,2,opt,name=password"`
	NetworkName string `json:"networkName,omitempty" protobuf:"bytes,3,opt,name=networkName"`
}

type AzureSpec struct {
	InstanceImageVersion string `json:"instanceImageVersion,omitempty" protobuf:"bytes,1,opt,name=instanceImageVersion"`
	RootPassword         string `json:"rootPassword,omitempty" protobuf:"bytes,2,opt,name=rootPassword"`
	SubnetCIDR           string `json:"subnetCidr,omitempty" protobuf:"bytes,3,opt,name=subnetCidr"`
	ResourceGroup        string `json:"resourceGroup,omitempty" protobuf:"bytes,4,opt,name=resourceGroup"`
	SubnetName           string `json:"subnetName,omitempty" protobuf:"bytes,5,opt,name=subnetName"`
	SecurityGroupName    string `json:"securityGroupName,omitempty" protobuf:"bytes,6,opt,name=securityGroupName"`
	VnetName             string `json:"vnetName,omitempty" protobuf:"bytes,7,opt,name=vnetName"`
	RouteTableName       string `json:"routeTableName,omitempty" protobuf:"bytes,8,opt,name=routeTableName"`
	StorageAccountName   string `json:"azureStorageAccountName,omitempty" protobuf:"bytes,9,opt,name=azureStorageAccountName"`
}

type LinodeSpec struct {
	// Linode
	RootPassword string `json:"rootPassword,omitempty" protobuf:"bytes,1,opt,name=rootPassword"`
	KernelId     int64  `json:"kernelId,omitempty" protobuf:"varint,2,opt,name=kernelId"`
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
	SShKeyExternalID string `json:"sshKeyExternalID,omitempty" protobuf:"bytes,1,opt,name=sshKeyExternalID"`
	//AWS              *AWSStatus `json:"aws,omitempty" protobuf:"bytes,2,opt,name=aws"`
}

type PharmerClusterStatus struct {
	Phase  ClusterPhase `json:"phase,omitempty,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=ClusterPhase"`
	Reason string       `json:"reason,omitempty,omitempty" protobuf:"bytes,2,opt,name=reason"`
	Cloud  CloudStatus  `json:"cloud,omitempty" protobuf:"bytes,4,opt,name=cloud"`
	//ReservedIPs  []ReservedIP       `json:"reservedIP,omitempty" protobuf:"bytes,6,rep,name=reservedIP"`
}

type ReservedIP struct {
	IP   string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`
	ID   string `json:"id,omitempty" protobuf:"bytes,2,opt,name=id"`
	Name string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
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
