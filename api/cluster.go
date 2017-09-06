package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/crypto/rand"
	. "github.com/appscode/go/encoding/json/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

// ref: https://github.com/kubernetes/kubernetes/blob/8b9f0ea5de2083589f3b9b289b90273556bc09c4/pkg/cloudprovider/providers/azure/azure.go#L56
type AzureCloudConfig struct {
	TenantID           string `json:"tenantId,omitempty"`
	SubscriptionID     string `json:"subscriptionId,omitempty"`
	AadClientID        string `json:"aadClientId,omitempty"`
	AadClientSecret    string `json:"aadClientSecret,omitempty"`
	ResourceGroup      string `json:"resourceGroup,omitempty"`
	Location           string `json:"location,omitempty"`
	SubnetName         string `json:"subnetName,omitempty"`
	SecurityGroupName  string `json:"securityGroupName,omitempty"`
	VnetName           string `json:"vnetName,omitempty"`
	RouteTableName     string `json:"routeTableName,omitempty"`
	StorageAccountName string `json:"storageAccountName,omitempty"`
}

// ref: https://github.com/kubernetes/kubernetes/blob/8b9f0ea5de2083589f3b9b289b90273556bc09c4/pkg/cloudprovider/providers/gce/gce.go#L228
type GCECloudConfig struct {
	TokenURL           string   `gcfg:"token-url"            ini:"token-url,omitempty"`
	TokenBody          string   `gcfg:"token-body"           ini:"token-body,omitempty"`
	ProjectID          string   `gcfg:"project-id"           ini:"project-id,omitempty"`
	NetworkName        string   `gcfg:"network-name"         ini:"network-name,omitempty"`
	NodeTags           []string `gcfg:"node-tags"            ini:"node-tags,omitempty,omitempty"`
	NodeInstancePrefix string   `gcfg:"node-instance-prefix" ini:"node-instance-prefix,omitempty,omitempty"`
	Multizone          bool     `gcfg:"multizone"            ini:"multizone,omitempty"`
}

type IG struct {
	SKU           string `json:"sku,omitempty"`
	Count         int64  `json:"count,omitempty"`
	SpotInstances bool   `json:"spotInstances,omitempty"`
}

type Cluster struct {
	metav1.TypeMeta   `json:",inline,omitempty,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitempty"`
	Spec              ClusterSpec   `json:"spec,omitempty,omitempty"`
	Status            ClusterStatus `json:"status,omitempty,omitempty"`
}

type Networking struct {
	PodSubnet     string `json:"podSubnet,omitempty"`
	ServiceSubnet string `json:"serviceSubnet,omitempty"`
	DNSDomain     string `json:"dnsDomain,omitempty"`

	// NEW
	NetworkProvider string `json:"networkProvider,omitempty"` // kubenet, flannel, calico, opencontrail
	// Replacing API_SERVERS https://github.com/kubernetes/kubernetes/blob/62898319dff291843e53b7839c6cde14ee5d2aa4/cluster/aws/util.sh#L1004
	DNSServerIP       string `json:"dnsServerIP,omitempty"`
	NonMasqueradeCIDR string `json:"nonMasqueradeCIDR,omitempty"`

	MasterSubnet string `json:"masterSubnet,omitempty"` // delete ?
}

type AWSSpec struct {
	// aws:TAG KubernetesCluster => clusterid
	IAMProfileMaster string `json:"iamProfileMaster,omitempty"`
	IAMProfileNode   string `json:"iamProfileNode,omitempty"`
	MasterSGName     string `json:"masterSGName,omitempty"`
	NodeSGName       string `json:"nodeSGName,omitempty"`

	VpcCIDR        string `json:"vpcCIDR,omitempty"`
	VpcCIDRBase    string `json:"vpcCIDRBase,omitempty"`
	MasterIPSuffix string `json:"masterIPSuffix,omitempty"`
	SubnetCIDR     string `json:"subnetCidr,omitempty"`
}

type GoogleSpec struct {
	// gce
	// NODE_SCOPES="${NODE_SCOPES:-compute-rw,monitoring,logging-write,storage-ro}"
	NodeScopes []string `json:"nodeScopes,omitempty"`
	// instance means either master or node
	CloudConfig *GCECloudConfig `json:"gceCloudConfig,omitempty"`
}

type AzureSpec struct {
	StorageAccountName string `json:"azureStorageAccountName,omitempty"`
	//only Azure
	InstanceImageVersion string            `json:"instanceImageVersion,omitempty"`
	CloudConfig          *AzureCloudConfig `json:"azureCloudConfig,omitempty"`
	InstanceRootPassword string            `json:"instanceRootPassword,omitempty"`
	SubnetCIDR           string            `json:"subnetCidr,omitempty"`
}

type LinodeSpec struct {
	// Azure, Linode
	InstanceRootPassword string `json:"instanceRootPassword,omitempty"`
}

type CloudSpec struct {
	CloudProvider   string `json:"cloudProvider,omitempty"`
	Project         string `json:"project,omitempty"`
	Region          string `json:"region,omitempty"`
	Zone            string `json:"zone,omitempty"` // master needs it for ossec
	OS              string `json:"os,omitempty"`
	Kernel          string `json:"kernel,omitempty"` // needed ?
	CloudConfigPath string `json:"cloudConfig,omitempty"`

	InstanceImage        string `json:"instanceImage,omitempty"`
	InstanceImageProject string `json:"instanceImageProject,omitempty"`

	AWS    *AWSSpec    `json:"aws,omitempty"`
	GCE    *GoogleSpec `json:"gce,omitempty"`
	Azure  *AzureSpec  `json:"azure,omitempty"`
	Linode *LinodeSpec `json:"linode,omitempty"`
}

type ClusterSpec struct {
	Cloud CloudSpec `json:"cloud"`

	API kubeadm.API `json:"api"`
	// move to api ?
	// 	KubeAPIserverRequestTimeout string `json:"kubeAPIserverRequestTimeout,omitempty"`
	// TerminatedPodGcThreshold    string `json:"terminatedPodGCThreshold,omitempty"`

	Etcd       kubeadm.Etcd `json:"etcd"`
	Networking Networking   `json:"networking"`

	Multizone StrToBool `json:"multizone,omitempty"`

	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	KubeadmVersion    string `json:"kubeadmVersion,omitempty"`

	// request data. This is needed to give consistent access to these values for all commands.
	DoNotDelete        bool     `json:"doNotDelete,omitempty"`
	AuthorizationModes []string `json:"authorizationModes,omitempty"`

	Token    string        `json:"token"`
	TokenTTL time.Duration `json:"tokenTTL"`

	// SelfHosted enables an alpha deployment type where the apiserver, scheduler, and
	// controller manager are managed by Kubernetes itself. This option is likely to
	// become the default in the future.
	SelfHosted bool `json:"selfHosted"`

	// APIServerCertSANs sets extra Subject Alternative Names for the API Server signing cert
	APIServerCertSANs     []string `json:"apiServerCertSANs,omitempty"`
	ClusterExternalDomain string   `json:"clusterExternalDomain,omitempty"`
	ClusterInternalDomain string   `json:"clusterInternalDomain,omitempty"`

	// Auto Set
	// https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/util.sh#L538
	CACertName           string `json:"caCertPHID,omitempty"`
	FrontProxyCACertName string `json:"frontProxyCaCertPHID,omitempty"`
	CredentialName       string `json:"credentialName,omitempty"`

	// Cloud Config

	AllocateNodeCIDRs            bool   `json:"allocateNodeCidrs,omitempty"`
	LoggingDestination           string `json:"loggingDestination,omitempty"`
	ElasticsearchLoggingReplicas int    `json:"elasticsearchLoggingReplicas,omitempty"`
	AdmissionControl             string `json:"admissionControl,omitempty"`
	RuntimeConfig                string `json:"runtimeConfig,omitempty"`

	// Kube 1.3
	AppscodeAuthnURL string `json:"appscodeAuthnURL,omitempty"`
	AppscodeAuthzURL string `json:"appscodeAuthzURL,omitempty"`

	// Kube 1.5.4

	// config
	// Some of these parameters might be useful to expose to users to configure as they please.
	// For now, use the default value used by the Kubernetes project as the default value.

	// TODO: Download the kube binaries from GCS bucket and ignore EU data locality issues for now.

	// common

	// GCE: Use Root Field for this in GCE

	// MASTER_TAG="clusterName-master"
	// NODE_TAG="clusterName-node"

	// aws
	// NODE_SCOPES=""

	// NEW
	// enable various v1beta1 features

	AutoscalerMinNodes    int     `json:"autoscalerMinNodes,omitempty"`
	AutoscalerMaxNodes    int     `json:"autoscalerMaxNodes,omitempty"`
	TargetNodeUtilization float64 `json:"targetNodeUtilization,omitempty"`

	// only aws

	EnableClusterMonitoring string `json:"enableClusterMonitoring,omitempty"`
	EnableClusterLogging    bool   `json:"enableClusterLogging,omitempty"`
	EnableNodeLogging       bool   `json:"enableNodeLogging,omitempty"`
	EnableCustomMetrics     string `json:"enableCustomMetrics,omitempty"`
	// NEW
	EnableAPIserverBasicAudit bool   `json:"enableAPIserverBasicAudit,omitempty"`
	EnableClusterAlert        string `json:"enableClusterAlert,omitempty"`
	EnableNodeProblemDetector bool   `json:"enableNodeProblemDetector,omitempty"`
	// Kub1 1.4
	EnableRescheduler                bool `json:"enableRescheduler,omitempty"`
	EnableWebhookTokenAuthentication bool `json:"enableWebhookTokenAuthn,omitempty"`
	EnableWebhookTokenAuthorization  bool `json:"enableWebhookTokenAuthz,omitempty"`
	EnableRBACAuthorization          bool `json:"enableRbacAuthz,omitempty"`
	EnableNodePublicIP               bool `json:"enableNodePublicIP,omitempty"`
	EnableNodeAutoscaler             bool `json:"enableNodeAutoscaler,omitempty"`

	// Consolidate DNS / Master name options
	// Deprecated
	KubernetesMasterName string `json:"kubernetesMasterName,omitempty"`
	// Deprecated
	MasterInternalIP string `json:"masterInternalIp,omitempty"`
	// the master root ebs volume size (typically does not need to be very large)
	// Deprecated
	MasterDiskId string `json:"masterDiskID,omitempty"`

	// Delete since moved to InstanceGroup / Instance
	// Deprecated
	MasterDiskType string `json:"masterDiskType,omitempty"`
	// Deprecated
	MasterDiskSize int64 `json:"masterDiskSize,omitempty"`
	// Deprecated
	MasterSKU string `json:"masterSku,omitempty"`
	// If set to Elasticsearch IP, master instance will be associated with this IP.
	// If set to auto, a new Elasticsearch IP will be acquired
	// Otherwise amazon-given public ip will be used (it'll change with reboot).
	// Deprecated
	MasterReservedIP string `json:"masterReservedIp,omitempty"`
	// Deprecated
	MasterExternalIP string `json:"masterExternalIp,omitempty"`

	// the node root ebs volume size (used to house docker images)
	// Deprecated
	NodeDiskType string `json:"nodeDiskType,omitempty"`
	// Deprecated
	NodeDiskSize int64 `json:"nodeDiskSize,omitempty"`
}

type AWSStatus struct {
	MasterSGId string `json:"masterSGID,omitempty"`
	NodeSGId   string `json:"nodeSGID,omitempty"`

	VpcId         string `json:"vpcID,omitempty"`
	SubnetId      string `json:"subnetID,omitempty"`
	RouteTableId  string `json:"routeTableID,omitempty"`
	IGWId         string `json:"igwID,omitempty"`
	DHCPOptionsId string `json:"dhcpOptionsID,omitempty"`

	// only aws
	RootDeviceName string `json:"-"`
}

type CloudStatus struct {
	AWS *AWSStatus `json:"aws,omitempty"`
}

type ClusterStatus struct {
	Phase            string `json:"phase,omitempty,omitempty"`
	Reason           string `json:"reason,omitempty,omitempty"`
	SSHKeyExternalID string `json:"sshKeyExternalID,omitempty"`

	Cloud CloudStatus `json:"cloud"`
}

func (cluster *Cluster) NodeCount() int64 {
	panic("Fix it!")
	return 0
}

func (cluster *Cluster) SetNodeGroups(ng []*proto.InstanceGroup) {
	panic("Fix it!")
}

func (cluster *Cluster) Delete() error {
	if cluster.Status.Phase == ClusterPhasePending || cluster.Status.Phase == ClusterPhaseFailing || cluster.Status.Phase == ClusterPhaseFailed {
		cluster.Status.Phase = ClusterPhaseFailed
	} else {
		cluster.Status.Phase = ClusterPhaseDeleted
	}
	fmt.Println("FixIt!")
	//if err := ctx.Save(); err != nil {
	//	return err
	//}

	n := rand.WithUniqSuffix(cluster.Name)
	//if _, err := ctx.Store().Engine.Update(&Kubernetes{Name: n}, &Kubernetes{PHID: ctx.PHID}); err != nil {
	//	return err
	//}
	cluster.Name = n
	return nil
}

func (cluster *Cluster) clusterIP(seq int64) string {
	octets := strings.Split(cluster.Spec.Networking.ServiceSubnet, ".")
	p, _ := strconv.ParseInt(octets[3], 10, 64)
	p = p + seq
	octets[3] = strconv.FormatInt(p, 10)
	return strings.Join(octets, ".")
}

func (cluster *Cluster) KubernetesClusterIP() string {
	return cluster.clusterIP(1)
}

// This is a onetime initializer method.
func (cluster *Cluster) APIServerURL() string {
	//if ctx.ApiServerUrl == "" {
	//	host := ctx.Extra().ExternalDomain(ctx.Name)
	//	if ctx.MasterReservedIP != "" {
	//		host = ctx.MasterReservedIP
	//	}
	return fmt.Sprintf("https://%v:6443", cluster.Spec.MasterReservedIP)
	// ctx.Logger().Infoln(fmt.Sprintf("Cluster %v 's api server url: %v\n", ctx.Name, ctx.ApiServerUrl))
	//}
}
