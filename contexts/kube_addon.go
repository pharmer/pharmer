package contexts

import (
	"fmt"

	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/contexts/auth"
	"github.com/appscode/pharmer/system"
	grpcContext "golang.org/x/net/context"
)

type KubeAddonContext struct {
	Zone string `json:"ZONE"` // master needs it for ossec

	// Replacing API_SERVERS https://github.com/kubernetes/kubernetes/blob/62898319dff291843e53b7839c6cde14ee5d2aa4/cluster/aws/util.sh#L1004
	KubernetesMasterName string `json:"KUBERNETES_MASTER_NAME"`
	MasterInternalIP     string `json:"MASTER_INTERNAL_IP"`

	EnableClusterMonitoring string `json:"ENABLE_CLUSTER_MONITORING"`
	EnableClusterSecurity   string `json:"ENABLE_CLUSTER_SECURITY"`
	EnableClusterLogging    bool   `json:"ENABLE_CLUSTER_LOGGING"`
	EnableNodeLogging       bool   `json:"ENABLE_NODE_LOGGING"`
	LoggingDestination      string `json:"LOGGING_DESTINATION"`
	EnableClusterDNS        bool   `json:"ENABLE_CLUSTER_DNS"`
	EnableClusterRegistry   bool   `json:"ENABLE_CLUSTER_REGISTRY"`
	DNSDomain               string `json:"DNS_DOMAIN"`
	CaCert                  string `json:"CA_CERT"`

	EnableClusterVpn string `json:"ENABLE_CLUSTER_VPN"`

	//ClusterName
	//  NodeInstancePrefix
	Name       string `json:"INSTANCE_PREFIX"`
	BucketName string `json:"BUCKET_NAME, omitempty"`

	// NEW
	NetworkProvider string `json:"NETWORK_PROVIDER"` // opencontrail, flannel, kubenet, calico, none

	EnvTimestamp string `json:"ENV_TIMESTAMP"`

	// NEW
	EnableClusterAlert string `json:"ENABLE_CLUSTER_ALERT"`

	Provider string `json:"PROVIDER"`
	OS       string `json:"OS"`
	Kernel   string `json:"Kernel"`

	// Kube 1.3
	PHID                      string `json:"KUBE_UID"`
	EnableNodeProblemDetector bool   `json:"ENABLE_NODE_PROBLEM_DETECTOR"`
	NetworkPolicyProvider     string `json:"NETWORK_POLICY_PROVIDER"` // calico

	// Kub1 1.4
	EnableRescheduler bool `json:"ENABLE_RESCHEDULER"`

	KubeUser        string `json:"KUBE_USER"`
	KubePassword    string `json:"KUBE_PASSWORD"`
	KubeBearerToken string `json:"KUBE_BEARER_TOKEN"`

	// NEW
	// APPSCODE ONLY
	AppsCodeNamespace         string `json:"APPSCODE_NS"`
	AppsCodeClusterUser       string `json:"APPSCODE_CLUSTER_USER"` // used by icinga, daemon
	AppsCodeApiToken          string `json:"APPSCODE_API_TOKEN"`    // used by icinga, daemon
	AppsCodeClusterRootDomain string `json:"APPSCODE_CLUSTER_ROOT_DOMAIN"`

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

	// APPSCODE ONLY

	// common
	*context    `json:"-"`
	*kubeClient `json:"-"`

	// request data. This is needed to give consistent access to these values for all commands.
	Region              string            `json:"REGION"`
	MasterSKU           string            `json:"MASTER_SKU"`
	NodeSet             map[string]int64  `json:"NODE_SET"`
	NodeGroups          []*InstanceGroup  `json:"NODE_GROUPS"`
	CloudCredentialPHID string            `json:"CLOUD_CREDENTIAL_PHID"`
	CloudCredential     map[string]string `json:"-"`
	Status              string            `json:"-"`
	StatusCause         string            `json:"-"`
	Sku                 string            `json:"-"`

	KubeVersion        string `json:"KUBE_VERSION"`
	KubeServerVersion  string `json:"KUBE_SERVER_VERSION"`
	SaltbaseVersion    string `json:"SALTBASE_VERSION"`
	KubeStarterVersion string `json:"KUBE_STARTER_VERSION"`
	HostfactsVersion   string `json:"HOSTFACTS_VERSION"`

	AppsCodeLogIndexPrefix            string `json:"APPSCODE_LOG_INDEX_PREFIX"`
	AppsCodeLogStorageLifetime        int64  `json:"APPSCODE_LOG_STORAGE_LIFETIME"`
	AppsCodeMonitoringStorageLifetime int64  `json:"APPSCODE_MONITORING_STORAGE_LIFETIME"`

	// If set to Elasticsearch IP, master instance will be associated with this IP.
	// If set to auto, a new Elasticsearch IP will be acquired
	// Otherwise amazon-given public ip will be used (it'll change with reboot).
	MasterReservedIP string `json:"MASTER_RESERVED_IP"`
	MasterExternalIP string `json:"MASTER_EXTERNAL_IP"`
	ApiServerUrl     string `json:"API_SERVER_URL"`

	// https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/util.sh#L538
	CaCertPHID string `json:"CA_CERT_PHID"`

	// Context Version is assigned on insert. If you want to force new version, set this value to 0 and call ctx.Save()
	ContextVersion int64 `json:"-"`
	// only aws

	// only GCE
	Project string `json:"GCE_PROJECT"`

	// only Linode
	InstanceRootPassword string `json:"INSTANCE_ROOT_PASSWORD"`

	// Cloud Config
	CloudConfigPath  string                `json:"CLOUD_CONFIG"`
	AzureCloudConfig *api.AzureCloudConfig `json:"AZURE_CLOUD_CONFIG"`
	GCECloudConfig   *api.GCECloudConfig   `json:"GCE_CLOUD_CONFIG"`
}

func NewKubeAddonContextFromAuth(a *auth.AuthInfo, uid string) (*KubeAddonContext, error) {
	return NewKubeAddonContext(NewBackgroundContext(a), uid)
}

func NewKubeAddonContext(c grpcContext.Context, uid string) (*KubeAddonContext, error) {
	return nil, nil
}

// This is a onetime initializer method.
func (ctx *KubeAddonContext) DetectApiServerURL() {
	if ctx.ApiServerUrl == "" {
		host := system.ClusterExternalDomain(ctx.Auth.Namespace, ctx.Name)
		if ctx.MasterReservedIP != "" {
			host = ctx.MasterReservedIP
		}
		ctx.ApiServerUrl = fmt.Sprintf("https://%v:6443", host)
	}
}
