package api

import (
	"encoding/json"
	"fmt"
	"os/user"
	"strconv"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	ssh "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	. "github.com/appscode/go/encoding/json/types"
	_env "github.com/appscode/go/env"
	"github.com/golang/protobuf/jsonpb"
	"github.com/zabawaba99/fireauth"
)

type AzureCloudConfig struct {
	TenantID           string `json:"tenantId"`
	SubscriptionID     string `json:"subscriptionId"`
	AadClientID        string `json:"aadClientId"`
	AadClientSecret    string `json:"aadClientSecret"`
	ResourceGroup      string `json:"resourceGroup"`
	Location           string `json:"location"`
	SubnetName         string `json:"subnetName"`
	SecurityGroupName  string `json:"securityGroupName"`
	VnetName           string `json:"vnetName"`
	RouteTableName     string `json:"routeTableName"`
	StorageAccountName string `json:"storageAccountName"`
}

type GCECloudConfig struct {
	TokenURL           string   `gcfg:"token-url"            ini:"token-url"`
	TokenBody          string   `gcfg:"token-body"           ini:"token-body"`
	ProjectID          string   `gcfg:"project-id"           ini:"project-id"`
	NetworkName        string   `gcfg:"network-name"         ini:"network-name"`
	NodeTags           []string `gcfg:"node-tags"            ini:"node-tags,omitempty"`
	NodeInstancePrefix string   `gcfg:"node-instance-prefix" ini:"node-instance-prefix,omitempty"`
	Multizone          bool     `gcfg:"multizone"            ini:"multizone"`
}

type MasterKubeEnv struct {
	//KubeUser        string `json:"KUBE_USER"`
	//KubePassword    string `json:"KUBE_PASSWORD"`
	KubeBearerToken string `json:"KUBE_BEARER_TOKEN"`
	MasterCert      string `json:"MASTER_CERT"`
	MasterKey       string `json:"MASTER_KEY"`
	DefaultLBCert   string `json:"DEFAULT_LB_CERT"`
	DefaultLBKey    string `json:"DEFAULT_LB_KEY"`

	// PAIR
	RegisterMasterKubelet     bool `json:"REGISTER_MASTER_KUBELET"`
	RegisterMasterSchedulable bool `json:"REGISTER_MASTER_SCHEDULABLE"`
	// KubeletApiserver       string `json:"KUBELET_APISERVER"`

	// NEW
	EnableManifestUrl bool   `json:"ENABLE_MANIFEST_URL"`
	ManifestUrl       string `json:"MANIFEST_URL"`
	ManifestUrlHeader string `json:"MANIFEST_URL_HEADER"`
	// WARNING: NumNodes in deprecated. This is a hack used by Kubernetes to calculate amount of RAM
	// needed for various processes, like, kube apiserver, heapster. But this is also impossible to
	// change after cluster is provisioned. So, this field should not be used, instead use ClusterContext.NodeCount().
	// This field is left here, since it is used by salt stack at this time.
	NumNodes int64 `json:"NUM_NODES"`
	// NEW
	// APPSCODE ONLY
	AppsCodeApiGrpcEndpoint   string `json:"APPSCODE_API_GRPC_ENDPOINT"` // used by icinga, daemon
	AppsCodeApiHttpEndpoint   string `json:"APPSCODE_API_HTTP_ENDPOINT"` // used by icinga, daemon
	AppsCodeClusterUser       string `json:"APPSCODE_CLUSTER_USER"`      // used by icinga, daemon
	AppsCodeApiToken          string `json:"APPSCODE_API_TOKEN"`         // used by icinga, daemon
	AppsCodeClusterRootDomain string `json:"APPSCODE_CLUSTER_ROOT_DOMAIN"`
	AppsCodeClusterCreator    string `json:"APPSCODE_CLUSTER_CREATOR"` // Username used for Initial ClusterRoleBinding

	// Kube 1.3
	AppscodeAuthnUrl string `json:"APPSCODE_AUTHN_URL"`
	AppscodeAuthzUrl string `json:"APPSCODE_AUTHZ_URL"`

	// Kube 1.4
	StorageBackend string `json:"STORAGE_BACKEND"`

	// Kube 1.5.4
	EnableApiserverBasicAudit bool `json:"ENABLE_APISERVER_BASIC_AUDIT"`
	EnableAppscodeAttic       bool `json:"ENABLE_APPSCODE_ATTIC"`
}

func (k *MasterKubeEnv) SetDefaults() {
	k.EnableManifestUrl = false
	// TODO: FixIt!
	//k.AppsCodeApiGrpcEndpoint = system.PublicAPIGrpcEndpoint()
	//k.AppsCodeApiHttpEndpoint = system.PublicAPIHttpEndpoint()
	//k.AppsCodeClusterRootDomain = system.ClusterBaseDomain()

	k.StorageBackend = "etcd2"
	k.EnableApiserverBasicAudit = true
	k.EnableAppscodeAttic = true
}

type NodeKubeEnv struct {
	KubernetesContainerRuntime string `json:"CONTAINER_RUNTIME"`
	KubernetesConfigureCbr0    bool   `json:"KUBERNETES_CONFIGURE_CBR0"`
}

func (k *NodeKubeEnv) SetDefaults() {
	k.KubernetesContainerRuntime = "docker"
	k.KubernetesConfigureCbr0 = true
}

type CommonKubeEnv struct {
	Zone string `json:"ZONE"` // master needs it for ossec

	ClusterIPRange        string `json:"CLUSTER_IP_RANGE"`
	ServiceClusterIPRange string `json:"SERVICE_CLUSTER_IP_RANGE"`
	// Replacing API_SERVERS https://github.com/kubernetes/kubernetes/blob/62898319dff291843e53b7839c6cde14ee5d2aa4/cluster/aws/util.sh#L1004
	KubernetesMasterName  string `json:"KUBERNETES_MASTER_NAME"`
	MasterInternalIP      string `json:"MASTER_INTERNAL_IP"`
	ClusterExternalDomain string `json:"CLUSTER_EXTERNAL_DOMAIN"`
	ClusterInternalDomain string `json:"CLUSTER_INTERNAL_DOMAIN"`

	AllocateNodeCIDRs            bool   `json:"ALLOCATE_NODE_CIDRS"`
	EnableClusterMonitoring      string `json:"ENABLE_CLUSTER_MONITORING"`
	EnableClusterLogging         bool   `json:"ENABLE_CLUSTER_LOGGING"`
	EnableNodeLogging            bool   `json:"ENABLE_NODE_LOGGING"`
	LoggingDestination           string `json:"LOGGING_DESTINATION"`
	ElasticsearchLoggingReplicas int    `json:"ELASTICSEARCH_LOGGING_REPLICAS"`
	EnableClusterDNS             bool   `json:"ENABLE_CLUSTER_DNS"`
	EnableClusterRegistry        bool   `json:"ENABLE_CLUSTER_REGISTRY"`
	ClusterRegistryDisk          string `json:"CLUSTER_REGISTRY_DISK"`
	ClusterRegistryDiskSize      string `json:"CLUSTER_REGISTRY_DISK_SIZE"`
	DNSReplicas                  int    `json:"DNS_REPLICAS"`
	DNSServerIP                  string `json:"DNS_SERVER_IP"`
	DNSDomain                    string `json:"DNS_DOMAIN"`
	KubeProxyToken               string `json:"KUBE_PROXY_TOKEN"`
	KubeletToken                 string `json:"KUBELET_TOKEN"`
	AdmissionControl             string `json:"ADMISSION_CONTROL"`
	MasterIPRange                string `json:"MASTER_IP_RANGE"`
	RuntimeConfig                string `json:"RUNTIME_CONFIG"`
	CaCert                       string `json:"CA_CERT"`
	KubeletCert                  string `json:"KUBELET_CERT"`
	KubeletKey                   string `json:"KUBELET_KEY"`
	StartupConfigToken           string `json:"STARTUP_CONFIG_TOKEN"`

	//Kubeadm
	FrontProxyCaCert string `json:"FRONT_PROXY_CA_CERT"`
	CaKey            string `json:"CA_KEY"`
	FrontProxyCaKey  string `json:"FRONT_PROXY_CA_KEY"`
	UserCert         string `json:"USER_CERT"`
	UserKey          string `json:"USER_KEY"`

	EnableThirdPartyResource bool `json:"ENABLE_THIRD_PARTY_RESOURCE"`

	EnableClusterVPN string `json:"ENABLE_CLUSTER_VPN"`
	VpnPsk           string `json:"VPN_PSK"`

	// ref: https://github.com/appscode/searchlight/blob/master/docs/user-guide/hostfacts/deployment.md
	HostfactsAuthToken string `json:"HOSTFACTS_AUTH_TOKEN"`
	HostfactsCert      string `json:"HOSTFACTS_CERT"`
	HostfactsKey       string `json:"HOSTFACTS_KEY"`

	DockerStorage string `json:"DOCKER_STORAGE"`

	//ClusterName
	//  NodeInstancePrefix
	// Name       string `json:"INSTANCE_PREFIX"`
	BucketName string `json:"BUCKET_NAME, omitempty"`

	// NEW
	NetworkProvider string `json:"NETWORK_PROVIDER"` // opencontrail, flannel, kubenet, calico, none
	HairpinMode     string `json:"HAIRPIN_MODE"`     // promiscuous-bridge, hairpin-veth, none

	EnvTimestamp string `json:"ENV_TIMESTAMP"`

	// TODO: Needed if we build custom Kube image.
	// KubeImageTag       string `json:"KUBE_IMAGE_TAG"`
	KubeDockerRegistry string    `json:"KUBE_DOCKER_REGISTRY"`
	Multizone          StrToBool `json:"MULTIZONE"`
	NonMasqueradeCidr  string    `json:"NON_MASQUERADE_CIDR"`

	KubeletPort                 string `json:"KUBELET_PORT"`
	KubeApiserverRequestTimeout string `json:"KUBE_APISERVER_REQUEST_TIMEOUT"`
	TerminatedPodGcThreshold    string `json:"TERMINATED_POD_GC_THRESHOLD"`
	EnableCustomMetrics         string `json:"ENABLE_CUSTOM_METRICS"`
	// NEW
	EnableClusterAlert string `json:"ENABLE_CLUSTER_ALERT"`

	Provider string `json:"PROVIDER"`
	OS       string `json:"OS"`
	Kernel   string `json:"Kernel"`

	// Kube 1.3
	// PHID                      string `json:"KUBE_UID"`
	NodeLabels                string `json:"NODE_LABELS"`
	EnableNodeProblemDetector bool   `json:"ENABLE_NODE_PROBLEM_DETECTOR"`
	EvictionHard              string `json:"EVICTION_HARD"`

	ExtraDockerOpts       string `json:"EXTRA_DOCKER_OPTS"`
	FeatureGates          string `json:"FEATURE_GATES"`
	NetworkPolicyProvider string `json:"NETWORK_POLICY_PROVIDER"` // calico

	// Kub1 1.4
	EnableRescheduler bool `json:"ENABLE_RESCHEDULER"`

	EnableScheduledJobResource       bool `json:"ENABLE_SCHEDULED_JOB_RESOURCE"`
	EnableWebhookTokenAuthentication bool `json:"ENABLE_WEBHOOK_TOKEN_AUTHN"`
	EnableWebhookTokenAuthorization  bool `json:"ENABLE_WEBHOOK_TOKEN_AUTHZ"`
	EnableRBACAuthorization          bool `json:"ENABLE_RBAC_AUTHZ"`

	// Cloud Config
	CloudConfigPath  string            `json:"CLOUD_CONFIG"`
	AzureCloudConfig *AzureCloudConfig `json:"AZURE_CLOUD_CONFIG"`
	GCECloudConfig   *GCECloudConfig   `json:"GCE_CLOUD_CONFIG"`

	// Context Version is assigned on insert. If you want to force new version, set this value to 0 and call ctx.Save()
	ResourceVersion int64 `json:"RESOURCE_VERSION"`

	// https://linux-tips.com/t/what-is-kernel-soft-lockup/78
	SoftlockupPanic bool `json:"SOFTLOCKUP_PANIC"`
}

func (k *CommonKubeEnv) SetDefaults() error {
	if UseFirebase() {
		// Generate JWT token for Firebase Custom Auth
		// https://www.firebase.com/docs/rest/guide/user-auth.html#section-token-generation
		// https://github.com/zabawaba99/fireauth
		gen := fireauth.New("MAKE_IT_A_FLAG")
		fb, err := FirebaseUid()
		if err != nil {
			return errors.FromErr(err).Err()
		}
		data := fireauth.Data{"uid": fb}
		if err != nil {
			return errors.FromErr(err).Err()
		}
		token, err := gen.CreateToken(data, nil)
		if err != nil {
			return errors.FromErr(err).Err()
		}
		k.StartupConfigToken = token
	} else {
		k.StartupConfigToken = rand.Characters(128)
	}

	k.EnvTimestamp = time.Now().UTC().Format("20060102T15:04")
	k.ClusterIPRange = "10.244.0.0/16"
	k.AllocateNodeCIDRs = false
	k.EnableClusterMonitoring = "none"
	k.EnableClusterLogging = false
	k.EnableNodeLogging = false
	k.EnableClusterDNS = false
	k.EnableClusterRegistry = false

	k.EnableThirdPartyResource = true

	k.EnableClusterVPN = "none"
	k.VpnPsk = ""

	k.KubeDockerRegistry = "gcr.io/google_containers"

	k.EnableClusterAlert = "appscode"

	k.NetworkPolicyProvider = "none"
	k.EnableNodeProblemDetector = true

	k.EnableScheduledJobResource = true
	k.EnableWebhookTokenAuthentication = true
	k.EnableWebhookTokenAuthorization = false
	k.EnableRBACAuthorization = true
	k.SoftlockupPanic = true
	return nil
}

type KubeEnv struct {
	MasterKubeEnv
	NodeKubeEnv
	CommonKubeEnv
}

func (k *KubeEnv) SetDefaults() error {
	k.MasterKubeEnv.SetDefaults()
	k.NodeKubeEnv.SetDefaults()
	err := k.CommonKubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).Err()
	}
	if k.EnableWebhookTokenAuthentication {
		k.AppscodeAuthnUrl = "" // TODO: FixIt system.KuberntesWebhookAuthenticationURL()
	}
	if k.EnableWebhookTokenAuthorization {
		k.AppscodeAuthzUrl = "" // TODO: FixIt system.KuberntesWebhookAuthorizationURL()
	}
	return nil
}

type KubeStartupConfig struct {
	Role               string `json:"ROLE"`
	KubernetesMaster   bool   `json:"KUBERNETES_MASTER"`
	InitialEtcdCluster string `json:"INITIAL_ETCD_CLUSTER"`
}

type ClusterStartupConfig struct {
	KubeEnv
	KubeStartupConfig
}

func FirebaseUid() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", errors.FromErr(err).Err()
	}
	return usr.Username, nil // find a better username
}

func UseFirebase() bool {
	return _env.FromHost().DevMode() // TODO(tamal): FixIt!  && system.Config.SkipStartupConfigAPI
}

type InstanceGroup struct {
	Sku              string `json:"SKU"`
	Count            int64  `json:"COUNT"`
	UseSpotInstances bool   `json:"USE_SPOT_INSTANCES"`
}

type Cluster struct {
	TypeMeta   `json:",inline,omitempty"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       ClusterSpec   `json:"spec,omitempty"`
	Status     ClusterStatus `json:"status,omitempty"`
}

type ClusterSpec struct {
	KubeEnv

	// request data. This is needed to give consistent access to these values for all commands.
	Region              string            `json:"REGION"`
	MasterSKU           string            `json:"MASTER_SKU"`
	NodeSet             map[string]int64  `json:"NODE_SET"` // deprecated, use NODES
	NodeGroups          []*InstanceGroup  `json:"NODE_GROUPS"`
	CloudCredentialPHID string            `json:"CLOUD_CREDENTIAL_PHID"`
	CloudCredential     map[string]string `json:"-"`
	DoNotDelete         bool              `json:"-"`
	DefaultAccessLevel  string            `json:"-"`

	KubernetesVersion string `json:"KUBERNETES_VERSION"`

	// config
	// Some of these parameters might be useful to expose to users to configure as they please.
	// For now, use the default value used by the Kubernetes project as the default value.

	// TODO: Download the kube binaries from GCS bucket and ignore EU data locality issues for now.

	// common

	// the master root ebs volume size (typically does not need to be very large)
	MasterDiskType string `json:"MASTER_DISK_TYPE"`
	MasterDiskSize int64  `json:"MASTER_DISK_SIZE"`
	MasterDiskId   string `json:"MASTER_DISK_ID"`

	// the node root ebs volume size (used to house docker images)
	NodeDiskType string `json:"NODE_DISK_TYPE"`
	NodeDiskSize int64  `json:"NODE_DISK_SIZE"`

	// GCE: Use Root Field for this in GCE

	// MASTER_TAG="clusterName-master"
	// NODE_TAG="clusterName-node"

	// aws
	// NODE_SCOPES=""

	// gce
	// NODE_SCOPES="${NODE_SCOPES:-compute-rw,monitoring,logging-write,storage-ro}"
	NodeScopes        []string `json:"NODE_SCOPES"`
	PollSleepInterval int      `json:"POLL_SLEEP_INTERVAL"`

	// If set to Elasticsearch IP, master instance will be associated with this IP.
	// If set to auto, a new Elasticsearch IP will be acquired
	// Otherwise amazon-given public ip will be used (it'll change with reboot).
	MasterReservedIP string `json:"MASTER_RESERVED_IP"`
	MasterExternalIP string `json:"MASTER_EXTERNAL_IP"`
	ApiServerUrl     string `json:"API_SERVER_URL"`

	// NEW
	// enable various v1beta1 features

	EnableNodePublicIP bool `json:"ENABLE_NODE_PUBLIC_IP"`

	EnableNodeAutoscaler  bool    `json:"ENABLE_NODE_AUTOSCALER"`
	AutoscalerMinNodes    int     `json:"AUTOSCALER_MIN_NODES"`
	AutoscalerMaxNodes    int     `json:"AUTOSCALER_MAX_NODES"`
	TargetNodeUtilization float64 `json:"TARGET_NODE_UTILIZATION"`

	// instance means either master or node
	InstanceImage        string `json:"INSTANCE_IMAGE"`
	InstanceImageProject string `json:"INSTANCE_IMAGE_PROJECT"`

	// Generated data, always different or every cluster.

	ContainerSubnet string `json:"CONTAINER_SUBNET"` // TODO:where used?

	// https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/util.sh#L538
	CaCertPHID string `json:"CA_CERT_PHID"`

	//Kubeadm
	FrontProxyCaCertPHID string `json:"FRONT_PROXY_CA_CERT_PHID"`
	UserCertPHID         string `json:"USER_CERT_PHID"`
	KubeadmToken         string `json:"KUBEADM_TOKEN"`

	// only aws

	// Dynamically generated SSH key used for this cluster
	SSHKeyPHID       string      `json:"SSH_KEY_PHID"`
	SSHKey           *ssh.SSHKey `json:"-"`
	SSHKeyExternalID string      `json:"SSH_KEY_EXTERNAL_ID"`

	// aws:TAG KubernetesCluster => clusterid
	IAMProfileMaster string `json:"IAM_PROFILE_MASTER"`
	IAMProfileNode   string `json:"IAM_PROFILE_NODE"`
	MasterSGId       string `json:"MASTER_SG_ID"`
	MasterSGName     string `json:"MASTER_SG_NAME"`
	NodeSGId         string `json:"NODE_SG_ID"`
	NodeSGName       string `json:"NODE_SG_NAME"`

	VpcId          string `json:"VPC_ID"`
	VpcCidr        string `json:"VPC_CIDR"`
	VpcCidrBase    string `json:"VPC_CIDR_BASE"`
	MasterIPSuffix string `json:"MASTER_IP_SUFFIX"`
	SubnetId       string `json:"SUBNET_ID"`
	SubnetCidr     string `json:"SUBNET_CIDR"`
	RouteTableId   string `json:"ROUTE_TABLE_ID"`
	IGWId          string `json:"IGW_ID"`
	DHCPOptionsId  string `json:"DHCP_OPTIONS_ID"`

	// only GCE
	Project string `json:"GCE_PROJECT"`

	// only aws
	RootDeviceName string `json:"-"`

	//only Azure
	InstanceImageVersion    string `json:"INSTANCE_IMAGE_VERSION"`
	AzureStorageAccountName string `json:"AZURE_STORAGE_ACCOUNT_NAME"`

	// only Linode
	InstanceRootPassword string `json:"INSTANCE_ROOT_PASSWORD"`
}

type ClusterStatus struct {
	Phase  string `json:"phase,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (cluster *Cluster) SetNodeGroups(ng []*proto.InstanceGroup) {
	cluster.Spec.NodeGroups = make([]*InstanceGroup, len(ng))
	for i, g := range ng {
		cluster.Spec.NodeGroups[i] = &InstanceGroup{
			Sku:              g.Sku,
			Count:            g.Count,
			UseSpotInstances: g.UseSpotInstances,
		}
	}
}

//func (ctx *Cluster) AddEdge(src, dst string, typ ClusterOP) error {
//	return nil
//}

/*
func (ctx *ClusterContext) UpdateNodeCount() error {
	kv := &KubernetesVersion{ID: ctx.ContextVersion}
	hasCtxVersion, err := ctx.Store().Engine.Get(kv)
	if err != nil {
		return err
	}
	if !hasCtxVersion {
		return errors.New().WithCause(fmt.Errorf("Cluster %v is missing config version %v", ctx.Name, ctx.ContextVersion)).WithContext(ctx).Err()
	}

	jsonCtx, err := json.Marshal(ctx)
	if err != nil {
		return err
	}
	sc, err := ctx.Store().NewSecString(string(jsonCtx))
	if err != nil {
		return err
	}
	kv.Context, err = sc.Envelope()
	if err != nil {
		return err
	}
	_, err = ctx.Store().Engine.Id(kv.ID).Update(kv)
	if err != nil {
		return err
	}
	return nil
}
*/

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
func (cluster *Cluster) DetectApiServerURL() {
	panic("TODO: Remove this call")
	//if ctx.ApiServerUrl == "" {
	//	host := ctx.Extra().ExternalDomain(ctx.Name)
	//	if ctx.MasterReservedIP != "" {
	//		host = ctx.MasterReservedIP
	//	}
	//	ctx.ApiServerUrl = fmt.Sprintf("https://%v:6443", host)
	//	ctx.Logger().Infoln(fmt.Sprintf("Cluster %v 's api server url: %v\n", ctx.Name, ctx.ApiServerUrl))
	//}
}

func (cluster *Cluster) NodeCount() int64 {
	n := int64(0)
	if cluster.Spec.RegisterMasterKubelet {
		n = 1
	}
	for _, ng := range cluster.Spec.NodeGroups {
		n += ng.Count
	}
	return n
}

func (cluster *Cluster) StartupConfig(role string) *ClusterStartupConfig {
	var config ClusterStartupConfig
	config.KubeEnv = cluster.Spec.KubeEnv
	config.Role = role
	config.KubernetesMaster = role == RoleKubernetesMaster
	config.InitialEtcdCluster = cluster.Spec.KubernetesMasterName
	config.NumNodes = cluster.NodeCount()
	return &config
}

func (cluster *Cluster) StartupConfigJson(role string) (string, error) {
	confJson, err := json.Marshal(cluster.StartupConfig(role))
	if err != nil {
		return "", err
	}
	return string(confJson), nil
}

func (cluster *Cluster) StartupConfigResponse(role string) (string, error) {
	confJson, err := cluster.StartupConfigJson(role)
	if err != nil {
		return "", err
	}

	resp := &proto.ClusterStartupConfigResponse{
		Configuration: string(confJson),
	}
	m := jsonpb.Marshaler{}
	return m.MarshalToString(resp)
}

func (cluster *Cluster) NewInstances(matches func(i *Instance, md *InstanceMetadata) bool) (*ClusterInstances, error) {
	if matches == nil {
		return nil, errors.New(`Use "github.com/appscode/pharmer/cloud/lib".NewInstances`).Err()
	}
	return &ClusterInstances{
		matches:        matches,
		KubernetesPHID: cluster.UID,
		Instances:      make([]*Instance, 0),
	}, nil
}
