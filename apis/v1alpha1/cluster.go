package v1alpha1

import (
	"fmt"
	"strconv"
	"strings"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

const (
	ResourceCodeCluster = ""
	ResourceKindCluster = "Cluster"
	ResourceNameCluster = "cluster"
	ResourceTypeCluster = "clusters"
)

type VultrCloudConfig struct {
	Token string `json:"token,omitempty" protobuf:"bytes,1,opt,name=token"`
}

type LinodeCloudConfig struct {
	Token string `json:"token,omitempty" protobuf:"bytes,1,opt,name=token"`
	Zone  string `json:"zone,omitempty" protobuf:"bytes,2,opt,name=zone"`
}

type ScalewayCloudConfig struct {
	Organization string `json:"organization,omitempty" protobuf:"bytes,1,opt,name=organization"`
	Token        string `json:"token,omitempty" protobuf:"bytes,2,opt,name=token"`
	Region       string `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
}

type SoftlayerCloudConfig struct {
	UserName string `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
	ApiKey   string `json:"apiKey,omitempty" protobuf:"bytes,2,opt,name=apiKey"`
	Zone     string `json:"zone,omitempty" protobuf:"bytes,3,opt,name=zone"`
}

// ref: https://github.com/kubernetes/kubernetes/blob/8b9f0ea5de2083589f3b9b289b90273556bc09c4/pkg/cloudprovider/providers/azure/azure.go#L56
type AzureCloudConfig struct {
	TenantID           string `json:"tenantId,omitempty" protobuf:"bytes,1,opt,name=tenantId"`
	SubscriptionID     string `json:"subscriptionId,omitempty" protobuf:"bytes,2,opt,name=subscriptionId"`
	AadClientID        string `json:"aadClientId,omitempty" protobuf:"bytes,3,opt,name=aadClientId"`
	AadClientSecret    string `json:"aadClientSecret,omitempty" protobuf:"bytes,4,opt,name=aadClientSecret"`
	ResourceGroup      string `json:"resourceGroup,omitempty" protobuf:"bytes,5,opt,name=resourceGroup"`
	Location           string `json:"location,omitempty" protobuf:"bytes,6,opt,name=location"`
	SubnetName         string `json:"subnetName,omitempty" protobuf:"bytes,7,opt,name=subnetName"`
	SecurityGroupName  string `json:"securityGroupName,omitempty" protobuf:"bytes,8,opt,name=securityGroupName"`
	VnetName           string `json:"vnetName,omitempty" protobuf:"bytes,9,opt,name=vnetName"`
	RouteTableName     string `json:"routeTableName,omitempty" protobuf:"bytes,10,opt,name=routeTableName"`
	StorageAccountName string `json:"storageAccountName,omitempty" protobuf:"bytes,11,opt,name=storageAccountName"`
}

// ref: https://github.com/kubernetes/kubernetes/blob/8b9f0ea5de2083589f3b9b289b90273556bc09c4/pkg/cloudprovider/providers/gce/gce.go#L228
type GCECloudConfig struct {
	TokenURL           string   `gcfg:"token-url" ini:"token-url,omitempty" protobuf:"bytes,1,opt,name=tokenURL"`
	TokenBody          string   `gcfg:"token-body" ini:"token-body,omitempty" protobuf:"bytes,2,opt,name=tokenBody"`
	ProjectID          string   `gcfg:"project-id" ini:"project-id,omitempty" protobuf:"bytes,3,opt,name=projectID"`
	NetworkName        string   `gcfg:"network-name" ini:"network-name,omitempty" protobuf:"bytes,4,opt,name=networkName"`
	NodeTags           []string `gcfg:"node-tags" ini:"node-tags,omitempty,omitempty" protobuf:"bytes,5,rep,name=nodeTags"`
	NodeInstancePrefix string   `gcfg:"node-instance-prefix" ini:"node-instance-prefix,omitempty,omitempty" protobuf:"bytes,6,opt,name=nodeInstancePrefix"`
	Multizone          bool     `gcfg:"multizone" ini:"multizone,omitempty" protobuf:"varint,7,opt,name=multizone"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Cluster struct {
	metav1.TypeMeta   `json:",inline,omitempty,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ClusterSpec   `json:"spec,omitempty,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            ClusterStatus `json:"status,omitempty,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type Networking struct {
	NetworkProvider string `json:"networkProvider,omitempty" protobuf:"bytes,4,opt,name=networkProvider"` // kubenet, flannel, calico, opencontrail
	PodSubnet       string `json:"podSubnet,omitempty" protobuf:"bytes,1,opt,name=podSubnet"`

	ServiceSubnet string `json:"serviceSubnet,omitempty" protobuf:"bytes,2,opt,name=serviceSubnet"`
	DNSDomain     string `json:"dnsDomain,omitempty" protobuf:"bytes,3,opt,name=dnsDomain"`

	// NEW
	// Replacing API_SERVERS https://github.com/kubernetes/kubernetes/blob/62898319dff291843e53b7839c6cde14ee5d2aa4/cluster/aws/util.sh#L1004
	DNSServerIP       string `json:"dnsServerIP,omitempty" protobuf:"bytes,5,opt,name=dnsServerIP"`
	NonMasqueradeCIDR string `json:"nonMasqueradeCIDR,omitempty" protobuf:"bytes,6,opt,name=nonMasqueradeCIDR"`
	MasterSubnet      string `json:"masterSubnet,omitempty" protobuf:"bytes,7,opt,name=masterSubnet"` // delete ?
}

func (n *Networking) SetDefaults() {
	if n.ServiceSubnet == "" {
		n.ServiceSubnet = kubeadmapi.DefaultServicesSubnet
	}
	if n.DNSDomain == "" {
		n.DNSDomain = kubeadmapi.DefaultServiceDNSDomain
	}
	if n.PodSubnet == "" {
		// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#pod-network
		switch n.NetworkProvider {
		case "calico":
			n.PodSubnet = "192.168.0.0/16"
		case "flannel":
			n.PodSubnet = "10.244.0.0/16"
		}
	}
}

type AWSSpec struct {
	// aws:TAG KubernetesCluster => clusterid
	IAMProfileMaster string `json:"iamProfileMaster,omitempty" protobuf:"bytes,1,opt,name=iamProfileMaster"`
	IAMProfileNode   string `json:"iamProfileNode,omitempty" protobuf:"bytes,2,opt,name=iamProfileNode"`
	MasterSGName     string `json:"masterSGName,omitempty" protobuf:"bytes,3,opt,name=masterSGName"`
	NodeSGName       string `json:"nodeSGName,omitempty" protobuf:"bytes,4,opt,name=nodeSGName"`

	VpcCIDR        string `json:"vpcCIDR,omitempty" protobuf:"bytes,5,opt,name=vpcCIDR"`
	VpcCIDRBase    string `json:"vpcCIDRBase,omitempty" protobuf:"bytes,6,opt,name=vpcCIDRBase"`
	MasterIPSuffix string `json:"masterIPSuffix,omitempty" protobuf:"bytes,7,opt,name=masterIPSuffix"`
	SubnetCIDR     string `json:"subnetCidr,omitempty" protobuf:"bytes,8,opt,name=subnetCidr"`
}

type GoogleSpec struct {
	// gce
	// NODE_SCOPES="${NODE_SCOPES:-compute-rw,monitoring,logging-write,storage-ro}"
	NodeScopes []string `json:"nodeScopes,omitempty" protobuf:"bytes,1,rep,name=nodeScopes"`
}

type AzureSpec struct {
	StorageAccountName string `json:"azureStorageAccountName,omitempty" protobuf:"bytes,1,opt,name=azureStorageAccountName"`
	//only Azure
	InstanceImageVersion string `json:"instanceImageVersion,omitempty" protobuf:"bytes,2,opt,name=instanceImageVersion"`
	RootPassword         string `json:"rootPassword,omitempty" protobuf:"bytes,4,opt,name=rootPassword"`
	SubnetCIDR           string `json:"subnetCidr,omitempty" protobuf:"bytes,5,opt,name=subnetCidr"`
}

type LinodeSpec struct {
	// Linode
	RootPassword string `json:"rootPassword,omitempty" protobuf:"bytes,1,opt,name=rootPassword"`
	KernelId     int64  `json:"kernelId,omitempty" protobuf:"varint,2,opt,name=kernelId"`
}

type SoftlayerSpec struct {
	CloudConfig *SoftlayerCloudConfig `json:"softlayerCloudConfig,omitempty" protobuf:"bytes,1,opt,name=softlayerCloudConfig"`
}

type CloudSpec struct {
	CloudProvider string `json:"cloudProvider,omitempty" protobuf:"bytes,1,opt,name=cloudProvider"`
	Project       string `json:"project,omitempty" protobuf:"bytes,2,opt,name=project"`
	Region        string `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
	Zone          string `json:"zone,omitempty" protobuf:"bytes,4,opt,name=zone"` // master needs it for ossec
	InstanceImage string `json:"instanceImage,omitempty" protobuf:"bytes,8,opt,name=instanceImage"`
	// Deprecated
	OS                   string `json:"os,omitempty" protobuf:"bytes,5,opt,name=os"`
	InstanceImageProject string `json:"instanceImageProject,omitempty" protobuf:"bytes,9,opt,name=instanceImageProject"`

	AWS       *AWSSpec       `json:"aws,omitempty" protobuf:"bytes,10,opt,name=aws"`
	GCE       *GoogleSpec    `json:"gce,omitempty" protobuf:"bytes,11,opt,name=gce"`
	Azure     *AzureSpec     `json:"azure,omitempty" protobuf:"bytes,12,opt,name=azure"`
	Linode    *LinodeSpec    `json:"linode,omitempty" protobuf:"bytes,13,opt,name=linode"`
	Softlayer *SoftlayerSpec `json:"softlayer,omitempty" protobuf:"bytes,14,opt,name=softlayer"`
}

type API struct {
	// AdvertiseAddress sets the address for the API server to advertise.
	AdvertiseAddress string `json:"advertiseAddress" protobuf:"bytes,1,opt,name=advertiseAddress"`
	// BindPort sets the secure port for the API Server to bind to
	BindPort int32 `json:"bindPort" protobuf:"varint,2,opt,name=bindPort"`
}

type ClusterSpec struct {
	Cloud CloudSpec `json:"cloud" protobuf:"bytes,1,opt,name=cloud"`

	API API `json:"api" protobuf:"bytes,2,opt,name=api"`

	// Etcd       kubeadm.Etcd `json:"etcd" protobuf:"bytes,3,opt,name=etcd"`
	Networking Networking `json:"networking" protobuf:"bytes,4,opt,name=networking"`

	KubernetesVersion string `json:"kubernetesVersion,omitempty" protobuf:"bytes,6,opt,name=kubernetesVersion"`
	KubeletVersion    string `json:"kubeletVersion,omitempty" protobuf:"bytes,55,opt,name=kubeletVersion"`
	KubeadmVersion    string `json:"kubeadmVersion,omitempty" protobuf:"bytes,56,opt,name=kubeadmVersion"`
	DoNotDelete       bool   `json:"doNotDelete,omitempty" protobuf:"varint,8,opt,name=doNotDelete"`

	// Auto Set
	// https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/util.sh#L538
	CACertName           string `json:"caCertPHID,omitempty" protobuf:"bytes,14,opt,name=caCertPHID"`
	FrontProxyCACertName string `json:"frontProxyCaCertPHID,omitempty" protobuf:"bytes,15,opt,name=frontProxyCaCertPHID"`
	CredentialName       string `json:"credentialName,omitempty" protobuf:"bytes,16,opt,name=credentialName"`

	// Deprecated
	MasterInternalIP string `json:"masterInternalIp,omitempty" protobuf:"bytes,41,opt,name=masterInternalIp"`
	// the master root ebs volume size (typically does not need to be very large)
	// Deprecated
	MasterDiskId string `json:"masterDiskID,omitempty" protobuf:"bytes,42,opt,name=masterDiskID"`

	// Delete since moved to NodeGroup / Instance
	// Deprecated
	MasterDiskType string `json:"masterDiskType,omitempty" protobuf:"bytes,43,opt,name=masterDiskType"`
	// If set to Elasticsearch IP, master instance will be associated with this IP.
	// If set to auto, a new Elasticsearch IP will be acquired
	// Otherwise amazon-given public ip will be used (it'll change with reboot).
	// Deprecated
	MasterReservedIP string `json:"masterReservedIp,omitempty" protobuf:"bytes,46,opt,name=masterReservedIp"`

	KubeletExtraArgs           map[string]string `json:"kubeletExtraArgs,omitempty" protobuf:"bytes,50,rep,name=kubeletExtraArgs"`
	APIServerExtraArgs         map[string]string `json:"apiServerExtraArgs,omitempty" protobuf:"bytes,51,rep,name=apiServerExtraArgs"`
	ControllerManagerExtraArgs map[string]string `json:"controllerManagerExtraArgs,omitempty" protobuf:"bytes,52,rep,name=controllerManagerExtraArgs"`
	SchedulerExtraArgs         map[string]string `json:"schedulerExtraArgs,omitempty" protobuf:"bytes,53,rep,name=schedulerExtraArgs"`
	AuthorizationModes         []string          `json:"authorizationModes,omitempty" protobuf:"bytes,54,rep,name=authorizationModes"`
	// APIServerCertSANs sets extra Subject Alternative Names for the API Server signing cert
	APIServerCertSANs []string `json:"apiServerCertSANs,omitempty" protobuf:"bytes,11,rep,name=apiServerCertSANs"`
}

type AWSStatus struct {
	MasterSGId string `json:"masterSGID,omitempty" protobuf:"bytes,1,opt,name=masterSGID"`
	NodeSGId   string `json:"nodeSGID,omitempty" protobuf:"bytes,2,opt,name=nodeSGID"`

	VpcId         string `json:"vpcID,omitempty" protobuf:"bytes,3,opt,name=vpcID"`
	SubnetId      string `json:"subnetID,omitempty" protobuf:"bytes,4,opt,name=subnetID"`
	RouteTableId  string `json:"routeTableID,omitempty" protobuf:"bytes,5,opt,name=routeTableID"`
	IGWId         string `json:"igwID,omitempty" protobuf:"bytes,6,opt,name=igwID"`
	DHCPOptionsId string `json:"dhcpOptionsID,omitempty" protobuf:"bytes,7,opt,name=dhcpOptionsID"`
	VolumeId      string `json:"volumeID,omitempty" protobuf:"bytes,8,opt,name=volumeID"`
	BucketName    string `json:"bucketName,omitempty" protobuf:"bytes,9,opt,name=bucketName"`

	// only aws
	RootDeviceName string `json:"-"`
}

type GCEStatus struct {
	BucketName string `json:"bucketName,omitempty" protobuf:"bytes,1,opt,name=bucketName"`
	// instance means either master or node
	CloudConfig *GCECloudConfig `json:"cloudConfig,omitempty" protobuf:"bytes,2,opt,name=gceCloudConfig"`
}

type AzureStatus struct {
	CloudConfig *AzureCloudConfig `json:"cloudConfig,omitempty" protobuf:"bytes,3,opt,name=azureCloudConfig"`
}

type VultrStatus struct {
	CloudConfig *VultrCloudConfig `json:"cloudConfig,omitempty" protobuf:"bytes,1,opt,name=vultrCloudConfig"`
}

type LinodeStatus struct {
	CloudConfig *LinodeCloudConfig `json:"cloudConfig,omitempty" protobuf:"bytes,1,opt,name=linodeCloudConfig"`
}

type ScalewayStatus struct {
	CloudConfig *ScalewayCloudConfig `json:"cloudConfig,omitempty" protobuf:"bytes,1,opt,name=scalewayCloudConfig"`
}

type CloudStatus struct {
	AWS      *AWSStatus      `json:"aws,omitempty" protobuf:"bytes,1,opt,name=aws"`
	GCE      *GCEStatus      `json:"gce,omitempty" protobuf:"bytes,2,opt,name=gce"`
	Azure    *AzureStatus    `json:"azure,omitempty" protobuf:"bytes,3,opt,name=azure"`
	Vultr    *VultrStatus    `json:"vultr,omitempty" protobuf:"bytes,4,opt,name=vultr"`
	Linode   *LinodeStatus   `json:"linode,omitempty" protobuf:"bytes,5,opt,name=linode"`
	Scaleway *ScalewayStatus `json:"scaleway,omitempty" protobuf:"bytes,6,opt,name=scaleway"`
}

/*
+---------------------------------+
|                                 |
|  +---------+     +---------+    |     +--------+
|  | PENDING +-----> FAILING +----------> FAILED |
|  +----+----+     +---------+    |     +--------+
|       |                         |
|       |                         |
|  +----v----+                    |
|  |  READY  |                    |
|  +----+----+                    |
|       |                         |
|       |                         |
|  +----v-----+                   |
|  | DELETING |                   |
|  +----+-----+                   |
|       |                         |
+---------------------------------+
        |
        |
   +----v----+
   | DELETED |
   +---------+
*/

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

type ClusterStatus struct {
	Phase            ClusterPhase       `json:"phase,omitempty,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=ClusterPhase"`
	Reason           string             `json:"reason,omitempty,omitempty" protobuf:"bytes,2,opt,name=reason"`
	SSHKeyExternalID string             `json:"sshKeyExternalID,omitempty" protobuf:"bytes,3,opt,name=sshKeyExternalID"`
	Cloud            CloudStatus        `json:"cloud,omitempty" protobuf:"bytes,4,opt,name=cloud"`
	APIAddresses     []core.NodeAddress `json:"apiServer,omitempty" protobuf:"bytes,5,rep,name=apiServer"`
	ReservedIPs      []ReservedIP       `json:"reservedIP,omitempty" protobuf:"bytes,6,rep,name=reservedIP"`
}

type ReservedIP struct {
	IP   string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`
	ID   string `json:"id,omitempty" protobuf:"bytes,2,opt,name=id"`
	Name string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
}

func (c *Cluster) clusterIP(seq int64) string {
	octets := strings.Split(c.Spec.Networking.ServiceSubnet, ".")
	p, _ := strconv.ParseInt(octets[3], 10, 64)
	p = p + seq
	octets[3] = strconv.FormatInt(p, 10)
	return strings.Join(octets, ".")
}

func (c *Cluster) KubernetesClusterIP() string {
	return c.clusterIP(1)
}

func (c Cluster) APIServerURL() string {
	m := map[core.NodeAddressType]string{}
	for _, addr := range c.Status.APIAddresses {
		m[addr.Type] = fmt.Sprintf("https://%s:%d", addr.Address, c.Spec.API.BindPort)
	}
	if u, found := m[core.NodeExternalIP]; found {
		return u
	}
	if u, found := m[core.NodeExternalDNS]; found {
		return u
	}
	return ""
}

func (c *Cluster) APIServerAddress() string {
	m := map[core.NodeAddressType]string{}
	for _, addr := range c.Status.APIAddresses {
		m[addr.Type] = fmt.Sprintf("%s:%d", addr.Address, c.Spec.API.BindPort)
	}
	if u, found := m[core.NodeExternalIP]; found {
		return u
	}
	if u, found := m[core.NodeInternalIP]; found {
		return u
	}
	if u, found := m[core.NodeHostName]; found {
		return u
	}
	if u, found := m[core.NodeInternalDNS]; found {
		return u
	}
	return ""
}

// Deprecated
type ClusterDeleteRequest struct {
	Name                 string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	ReleaseReservedIp    bool   `protobuf:"varint,2,opt,name=release_reserved_ip,json=releaseReservedIp" json:"release_reserved_ip,omitempty"`
	Force                bool   `protobuf:"varint,3,opt,name=force" json:"force,omitempty"`
	KeepLodabalancers    bool   `protobuf:"varint,4,opt,name=keep_lodabalancers,json=keepLodabalancers" json:"keep_lodabalancers,omitempty"`
	DeleteDynamicVolumes bool   `protobuf:"varint,5,opt,name=delete_dynamic_volumes,json=deleteDynamicVolumes" json:"delete_dynamic_volumes,omitempty"`
}
