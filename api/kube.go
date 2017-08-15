package api

import (
	"os/user"
	"time"

	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	. "github.com/appscode/go/encoding/json/types"
	_env "github.com/appscode/go/env"
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

	AppsCodeIcingaWebUser     string `json:"APPSCODE_ICINGA_WEB_USER"`
	AppsCodeIcingaWebPassword string `json:"APPSCODE_ICINGA_WEB_PASSWORD"`
	AppsCodeIcingaIdoUser     string `json:"APPSCODE_ICINGA_IDO_USER"`
	AppsCodeIcingaIdoPassword string `json:"APPSCODE_ICINGA_IDO_PASSWORD"`
	AppsCodeIcingaApiUser     string `json:"APPSCODE_ICINGA_API_USER"`
	AppsCodeIcingaApiPassword string `json:"APPSCODE_ICINGA_API_PASSWORD"`

	AppsCodeInfluxAdminUser     string `json:"APPSCODE_INFLUX_ADMIN_USER"`
	AppsCodeInfluxAdminPassword string `json:"APPSCODE_INFLUX_ADMIN_PASSWORD"`
	AppsCodeInfluxReadUser      string `json:"APPSCODE_INFLUX_READ_USER"`
	AppsCodeInfluxReadPassword  string `json:"APPSCODE_INFLUX_READ_PASSWORD"`
	AppsCodeInfluxWriteUser     string `json:"APPSCODE_INFLUX_WRITE_USER"`
	AppsCodeInfluxWritePassword string `json:"APPSCODE_INFLUX_WRITE_PASSWORD"`

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

	k.AppsCodeIcingaWebUser = "icingaweb"
	k.AppsCodeIcingaWebPassword = rand.GeneratePassword()
	k.AppsCodeIcingaIdoUser = "icingaido"
	k.AppsCodeIcingaIdoPassword = rand.GeneratePassword()
	k.AppsCodeIcingaApiUser = "icingaapi"
	k.AppsCodeIcingaApiPassword = rand.GeneratePassword()

	k.AppsCodeInfluxAdminUser = "acadmin"
	k.AppsCodeInfluxAdminPassword = rand.GeneratePassword()
	k.AppsCodeInfluxReadUser = "acreader"
	k.AppsCodeInfluxReadPassword = rand.GeneratePassword()
	k.AppsCodeInfluxWriteUser = "acwriter"
	k.AppsCodeInfluxWritePassword = rand.GeneratePassword()

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
	Name       string `json:"INSTANCE_PREFIX"`
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
	PHID                      string `json:"KUBE_UID"`
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
	ContextVersion int64 `json:"APPSCODE_CONTEXT_VERSION"`

	// Kubelet auth: https://kubernetes.io/docs/admin/kubelet-authentication-authorization/#kubelet-authorization
	KubeAPIServerCert string `json:"KUBE_API_SERVER_CERT"`
	KubeAPIServerKey  string `json:"KUBE_API_SERVER_KEY"`

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

type CommonNonEnv struct {
}

type ClusterStartupConfig struct {
	KubeEnv
	CommonNonEnv
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
