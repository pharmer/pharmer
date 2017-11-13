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

type PacketCloudConfig struct {
	Project string `json:"project,omitempty" protobuf:"bytes,1,opt,name=project"`
	ApiKey  string `json:"apiKey,omitempty" protobuf:"bytes,2,opt,name=apiKey"`
	Zone    string `json:"zone,omitempty" protobuf:"bytes,3,opt,name=zone"`
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
	NetworkProvider string `json:"networkProvider,omitempty"` // kubenet, flannel, calico, opencontrail
	PodSubnet       string `json:"podSubnet,omitempty"`
	ServiceSubnet   string `json:"serviceSubnet,omitempty"`
	DNSDomain       string `json:"dnsDomain,omitempty"`
	// NEW
	// Replacing API_SERVERS https://github.com/kubernetes/kubernetes/blob/62898319dff291843e53b7839c6cde14ee5d2aa4/cluster/aws/util.sh#L1004
	DNSServerIP       string `json:"dnsServerIP,omitempty"`
	NonMasqueradeCIDR string `json:"nonMasqueradeCIDR,omitempty"`
	MasterSubnet      string `json:"masterSubnet,omitempty"` // delete ?
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
	VpcCIDR          string `json:"vpcCIDR,omitempty" protobuf:"bytes,5,opt,name=vpcCIDR"`
	VpcCIDRBase      string `json:"vpcCIDRBase,omitempty" protobuf:"bytes,6,opt,name=vpcCIDRBase"`
	MasterIPSuffix   string `json:"masterIPSuffix,omitempty" protobuf:"bytes,7,opt,name=masterIPSuffix"`
	SubnetCIDR       string `json:"subnetCidr,omitempty" protobuf:"bytes,8,opt,name=subnetCidr"`
}

type GoogleSpec struct {
	NetworkName string   `gcfg:"network-name" ini:"network-name,omitempty"`
	NodeTags    []string `gcfg:"node-tags" ini:"node-tags,omitempty,omitempty"`
	// gce
	// NODE_SCOPES="${NODE_SCOPES:-compute-rw,monitoring,logging-write,storage-ro}"
	NodeScopes []string `json:"nodeScopes,omitempty"`
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

type CloudSpec struct {
	CloudProvider        string      `json:"cloudProvider,omitempty"`
	Project              string      `json:"project,omitempty"`
	Region               string      `json:"region,omitempty"`
	Zone                 string      `json:"zone,omitempty"` // master needs it for ossec
	InstanceImage        string      `json:"instanceImage,omitempty"`
	OS                   string      `json:"os,omitempty"`
	InstanceImageProject string      `json:"instanceImageProject,omitempty"`
	CCMCredentialName    string      `json:"ccmCredentialName,omitempty"`
	AWS                  *AWSSpec    `json:"aws,omitempty"`
	GCE                  *GoogleSpec `json:"gce,omitempty"`
	Azure                *AzureSpec  `json:"azure,omitempty"`
	Linode               *LinodeSpec `json:"linode,omitempty"`
}

type API struct {
	// AdvertiseAddress sets the address for the API server to advertise.
	AdvertiseAddress string `json:"advertiseAddress" protobuf:"bytes,1,opt,name=advertiseAddress"`
	// BindPort sets the secure port for the API Server to bind to
	BindPort int32 `json:"bindPort" protobuf:"varint,2,opt,name=bindPort"`
}

type ClusterSpec struct {
	Cloud                      CloudSpec         `json:"cloud"`
	API                        API               `json:"api"`
	Networking                 Networking        `json:"networking"`
	KubernetesVersion          string            `json:"kubernetesVersion,omitempty"`
	KubeletVersion             string            `json:"kubeletVersion,omitempty"`
	KubeadmVersion             string            `json:"kubeadmVersion,omitempty"`
	Locked                     bool              `json:"locked,omitempty"`
	CACertName                 string            `json:"caCertName,omitempty"`
	FrontProxyCACertName       string            `json:"frontProxyCACertName,omitempty"`
	CredentialName             string            `json:"credentialName,omitempty"`
	KubeletExtraArgs           map[string]string `json:"kubeletExtraArgs,omitempty"`
	APIServerExtraArgs         map[string]string `json:"apiServerExtraArgs,omitempty"`
	ControllerManagerExtraArgs map[string]string `json:"controllerManagerExtraArgs,omitempty"`
	SchedulerExtraArgs         map[string]string `json:"schedulerExtraArgs,omitempty"`
	AuthorizationModes         []string          `json:"authorizationModes,omitempty"`
	APIServerCertSANs          []string          `json:"apiServerCertSANs,omitempty"`

	// Deprecated
	MasterInternalIP string `json:"-"`
	// the master root ebs volume size (typically does not need to be very large)
	// Deprecated
	MasterDiskId string `json:"-"`

	// Delete since moved to NodeGroup / Instance
	// Deprecated
	MasterDiskType string `json:"-"`
	// If set to Elasticsearch IP, master instance will be associated with this IP.
	// If set to auto, a new Elasticsearch IP will be acquired
	// Otherwise amazon-given public ip will be used (it'll change with reboot).
	// Deprecated
	MasterReservedIP string `json:"-"`
}

type AWSStatus struct {
	MasterSGId string `json:"masterSGID,omitempty"`
	NodeSGId   string `json:"nodeSGID,omitempty"`

	VpcId         string `json:"vpcID,omitempty"`
	SubnetId      string `json:"subnetID,omitempty"`
	RouteTableId  string `json:"routeTableID,omitempty"`
	IGWId         string `json:"igwID,omitempty"`
	DHCPOptionsId string `json:"dhcpOptionsID,omitempty"`
	VolumeId      string `json:"volumeID,omitempty"`

	// Deprecated
	RootDeviceName string `json:"-"`
}

type CloudStatus struct {
	AWS *AWSStatus `json:"aws,omitempty" protobuf:"bytes,1,opt,name=aws"`
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
	Phase            ClusterPhase       `json:"phase,omitempty,omitempty"`
	Reason           string             `json:"reason,omitempty,omitempty"`
	SSHKeyExternalID string             `json:"sshKeyExternalID,omitempty"`
	Cloud            CloudStatus        `json:"cloud,omitempty"`
	APIAddresses     []core.NodeAddress `json:"apiServer,omitempty"`
	ReservedIPs      []ReservedIP       `json:"reservedIP,omitempty"`
}

type ReservedIP struct {
	IP   string `json:"ip,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
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

// ref: https://github.com/digitalocean/digitalocean-cloud-controller-manager#kubernetes-node-names-must-match-the-droplet-name
func (c *Cluster) APIServerAddress() string {
	m := map[core.NodeAddressType]string{}
	for _, addr := range c.Status.APIAddresses {
		m[addr.Type] = fmt.Sprintf("%s:%d", addr.Address, c.Spec.API.BindPort)
	}

	// ref: https://github.com/kubernetes/kubernetes/blob/d595003e0dc1b94455d1367e96e15ff67fc920fa/cmd/kube-apiserver/app/options/options.go#L99
	addrTypes := []core.NodeAddressType{
		core.NodeInternalDNS,
		core.NodeInternalIP,
		core.NodeExternalDNS,
		core.NodeExternalIP,
	}
	if pat, found := c.Spec.APIServerExtraArgs["kubelet-preferred-address-types"]; found {
		ats := strings.Split(pat, ",")
		addrTypes = make([]core.NodeAddressType, len(ats))
		for i, at := range ats {
			addrTypes[i] = core.NodeAddressType(at)
		}
	}

	for _, at := range addrTypes {
		if u, found := m[at]; found {
			return u
		}
	}
	return ""
}
