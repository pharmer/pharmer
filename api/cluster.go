package api

import (
	"fmt"
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	"github.com/appscode/go/crypto/rand"
	. "github.com/appscode/go/encoding/json/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	SKU              string `json:"sku,omitempty"`
	Count            int64  `json:"count,omitempty"`
	UseSpotInstances bool   `json:"useSpotInstances,omitempty"`
}

type Cluster struct {
	metav1.TypeMeta   `json:",inline,omitempty,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitempty"`
	Spec              ClusterSpec   `json:"spec,omitempty,omitempty"`
	Status            ClusterStatus `json:"status,omitempty,omitempty"`
}

type ClusterSpec struct {
	NodeGroups     []*IG  `json:"nodeGroups,omitempty"`
	CredentialName string `json:"credentialName,omitempty"`
	KubeadmToken   string `json:"kubeadmToken,omitempty"`

	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	KubeadmVersion    string `json:"kubeadmVersion,omitempty"`

	SSHKeyExternalID string `json:"sshKeyExternalID,omitempty"`

	// https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/util.sh#L538
	CACertName           string `json:"caCertPHID,omitempty"`
	FrontProxyCACertName string `json:"frontProxyCaCertPHID,omitempty"`
	AdminUserCertName    string `json:"userCertPHID,omitempty"`

	// request data. This is needed to give consistent access to these values for all commands.
	Region string `json:"region,omitempty"`

	DoNotDelete        bool   `json:"doNotDelete,omitempty"`
	DefaultAccessLevel string `json:"defaultAccessLevel,omitempty"`

	Zone string `json:"ZONE,omitempty"` // master needs it for ossec

	ClusterIPRange        string `json:"clusterIpRange,omitempty"`
	ServiceClusterIPRange string `json:"serviceClusterIpRange,omitempty"`
	// Replacing API_SERVERS https://github.com/kubernetes/kubernetes/blob/62898319dff291843e53b7839c6cde14ee5d2aa4/cluster/aws/util.sh#L1004
	ClusterExternalDomain string `json:"clusterExternalDomain,omitempty"`
	ClusterInternalDomain string `json:"clusterInternalDomain,omitempty"`

	MasterDiskType string `json:"masterDiskType,omitempty"`
	MasterDiskSize int64  `json:"masterDiskSize,omitempty"`

	MasterSKU            string `json:"masterSku,omitempty"`
	KubernetesMasterName string `json:"kubernetesMasterName,omitempty"`
	MasterInternalIP     string `json:"masterInternalIp,omitempty"`
	MasterIPRange        string `json:"masterIpRange,omitempty"`
	// the master root ebs volume size (typically does not need to be very large)
	MasterDiskId string `json:"masterDiskID,omitempty"`

	// If set to Elasticsearch IP, master instance will be associated with this IP.
	// If set to auto, a new Elasticsearch IP will be acquired
	// Otherwise amazon-given public ip will be used (it'll change with reboot).
	MasterReservedIP string `json:"masterReservedIp,omitempty"`
	MasterExternalIP string `json:"masterExternalIp,omitempty"`

	// the node root ebs volume size (used to house docker images)
	NodeDiskType string `json:"nodeDiskType,omitempty"`
	NodeDiskSize int64  `json:"nodeDiskSize,omitempty"`

	AllocateNodeCIDRs            bool   `json:"allocateNodeCidrs,omitempty"`
	EnableClusterMonitoring      string `json:"enableClusterMonitoring,omitempty"`
	EnableClusterLogging         bool   `json:"enableClusterLogging,omitempty"`
	EnableNodeLogging            bool   `json:"enableNodeLogging,omitempty"`
	LoggingDestination           string `json:"loggingDestination,omitempty"`
	ElasticsearchLoggingReplicas int    `json:"elasticsearchLoggingReplicas,omitempty"`
	DNSServerIP                  string `json:"dnsServerIP,omitempty"`
	DNSDomain                    string `json:"dnsDomain,omitempty"`
	AdmissionControl             string `json:"admissionControl,omitempty"`
	RuntimeConfig                string `json:"runtimeConfig,omitempty"`
	StartupConfigToken           string `json:"startupConfigToken,omitempty"`

	//ClusterName
	//  NodeInstancePrefix
	// Name       string `json:"INSTANCE_PREFIX,omitempty"`

	// NEW
	NetworkProvider string `json:"networkProvider,omitempty"` // opencontrail, flannel, kubenet, calico, none

	Multizone         StrToBool `json:"multizone,omitempty"`
	NonMasqueradeCIDR string    `json:"nonMasqueradeCIDR,omitempty"`

	KubeletPort                 string `json:"kubeletPort,omitempty"`
	KubeAPIserverRequestTimeout string `json:"kubeAPIserverRequestTimeout,omitempty"`
	TerminatedPodGcThreshold    string `json:"terminatedPodGCThreshold,omitempty"`
	EnableCustomMetrics         string `json:"enableCustomMetrics,omitempty"`
	// NEW
	EnableClusterAlert string `json:"enableClusterAlert,omitempty"`

	Provider string `json:"provider,omitempty"`
	OS       string `json:"os,omitempty"`
	Kernel   string `json:"kernel,omitempty"` // needed ?

	//NodeLabels                string `json:"nodeLabels,omitempty"`
	EnableNodeProblemDetector bool   `json:"enableNodeProblemDetector,omitempty"`
	NetworkPolicyProvider     string `json:"networkPolicyProvider,omitempty"` // calico

	// Kub1 1.4
	EnableRescheduler                bool `json:"enableRescheduler,omitempty"`
	EnableWebhookTokenAuthentication bool `json:"enableWebhookTokenAuthn,omitempty"`
	EnableWebhookTokenAuthorization  bool `json:"enableWebhookTokenAuthz,omitempty"`
	EnableRBACAuthorization          bool `json:"enableRbacAuthz,omitempty"`

	// Cloud Config
	CloudConfigPath  string            `json:"cloudConfig,omitempty"`
	AzureCloudConfig *AzureCloudConfig `json:"azureCloudConfig,omitempty"`
	GCECloudConfig   *GCECloudConfig   `json:"gceCloudConfig,omitempty"`

	// Kube 1.3
	AppscodeAuthnURL string `json:"appscodeAuthnURL,omitempty"`
	AppscodeAuthzURL string `json:"appscodeAuthzURL,omitempty"`

	// Kube 1.5.4
	EnableAPIserverBasicAudit bool `json:"enableAPIserverBasicAudit,omitempty"`

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

	// gce
	// NODE_SCOPES="${NODE_SCOPES:-compute-rw,monitoring,logging-write,storage-ro}"
	NodeScopes        []string `json:"nodeScopes,omitempty"`
	PollSleepInterval int      `json:"pollSleepInterval,omitempty"`

	// NEW
	// enable various v1beta1 features

	EnableNodePublicIP bool `json:"enableNodePublicIP,omitempty"`

	EnableNodeAutoscaler  bool    `json:"enableNodeAutoscaler,omitempty"`
	AutoscalerMinNodes    int     `json:"autoscalerMinNodes,omitempty"`
	AutoscalerMaxNodes    int     `json:"autoscalerMaxNodes,omitempty"`
	TargetNodeUtilization float64 `json:"targetNodeUtilization,omitempty"`

	// instance means either master or node
	InstanceImage        string `json:"instanceImage,omitempty"`
	InstanceImageProject string `json:"instanceImageProject,omitempty"`

	// Generated data, always different or every cluster.

	ContainerSubnet string `json:"containerSubnet,omitempty"` // TODO:where used?

	// only aws

	// aws:TAG KubernetesCluster => clusterid
	IAMProfileMaster string `json:"iamProfileMaster,omitempty"`
	IAMProfileNode   string `json:"iamProfileNode,omitempty"`
	MasterSGId       string `json:"masterSGID,omitempty"`
	MasterSGName     string `json:"masterSGName,omitempty"`
	NodeSGId         string `json:"nodeSGID,omitempty"`
	NodeSGName       string `json:"nodeSGName,omitempty"`

	VpcId          string `json:"vpcID,omitempty"`
	VpcCIDR        string `json:"vpcCIDR,omitempty"`
	VpcCIDRBase    string `json:"vpcCIDRBase,omitempty"`
	MasterIPSuffix string `json:"masterIPSuffix,omitempty"`
	SubnetId       string `json:"subnetID,omitempty"`
	SubnetCIDR     string `json:"subnetCidr,omitempty"`
	RouteTableId   string `json:"routeTableID,omitempty"`
	IGWId          string `json:"igwID,omitempty"`
	DHCPOptionsId  string `json:"dhcpOptionsID,omitempty"`

	// only GCE
	Project string `json:"gceProject,omitempty"`

	// only aws
	RootDeviceName string `json:"-"`

	//only Azure
	InstanceImageVersion    string `json:"instanceImageVersion,omitempty"`
	AzureStorageAccountName string `json:"azureStorageAccountName,omitempty"`

	// only Linode
	InstanceRootPassword string `json:"instanceRootPassword,omitempty"`
}

type ClusterStatus struct {
	Phase  string `json:"phase,omitempty,omitempty"`
	Reason string `json:"reason,omitempty,omitempty"`
}

func (cluster *Cluster) NodeCount() int64 {
	n := int64(1)
	for _, ng := range cluster.Spec.NodeGroups {
		n += ng.Count
	}
	return n
}

func (cluster *Cluster) SetNodeGroups(ng []*proto.InstanceGroup) {
	cluster.Spec.NodeGroups = make([]*IG, len(ng))
	for i, g := range ng {
		cluster.Spec.NodeGroups[i] = &IG{
			SKU:              g.Sku,
			Count:            g.Count,
			UseSpotInstances: g.UseSpotInstances,
		}
	}
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
	octets := strings.Split(cluster.Spec.ServiceClusterIPRange, ".")
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
