package api

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	ssh "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/go/crypto/rand"
	. "github.com/appscode/go/encoding/json/types"
	_env "github.com/appscode/go/env"
	"github.com/appscode/go/errors"
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
	MasterCert    string `json:"masterCert"`
	MasterKey     string `json:"masterKey"`
	DefaultLBCert string `json:"defaultLBCert"`
	DefaultLBKey  string `json:"defaultLBKey"`

	// PAIR
	RegisterMasterKubelet     bool `json:"registerMasterKubelet"`
	RegisterMasterSchedulable bool `json:"registerMasterSchedulable"`
	// KubeletApiserver       string `json:"KUBELET_APISERVER"`

	// NEW
	EnableManifestUrl bool   `json:"enableManifestURL"`
	ManifestUrl       string `json:"manifestURL"`
	ManifestUrlHeader string `json:"manifestURLHeader"`
	// WARNING: NumNodes in deprecated. This is a hack used by Kubernetes to calculate amount of RAM
	// needed for various processes, like, kube apiserver, heapster. But this is also impossible to
	// change after cluster is provisioned. So, this field should not be used, instead use ClusterContext.NodeCount().
	// This field is left here, since it is used by salt stack at this time.
	NumNodes int64 `json:"NUM_NODES"`

	// Kube 1.3
	AppscodeAuthnUrl string `json:"appscodeAuthnURL"`
	AppscodeAuthzUrl string `json:"appscodeAuthzURL"`

	// Kube 1.4
	StorageBackend string `json:"STORAGE_BACKEND"`

	// Kube 1.5.4
	EnableApiserverBasicAudit bool `json:"enableApiserverBasicAudit"`
	EnableAppscodeAttic       bool `json:"enableAppscodeAttic"`
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
	KubernetesContainerRuntime string `json:"containerRuntime"`
	KubernetesConfigureCbr0    bool   `json:"kubernetesConfigureCbr0"`
}

func (k *NodeKubeEnv) SetDefaults() {
	k.KubernetesContainerRuntime = "docker"
	k.KubernetesConfigureCbr0 = true
}

type CommonKubeEnv struct {
	Zone string `json:"ZONE"` // master needs it for ossec

	ClusterIPRange        string `json:"clusterIpRange"`
	ServiceClusterIPRange string `json:"serviceClusterIpRange"`
	// Replacing API_SERVERS https://github.com/kubernetes/kubernetes/blob/62898319dff291843e53b7839c6cde14ee5d2aa4/cluster/aws/util.sh#L1004
	KubernetesMasterName  string `json:"kubernetesMasterName"`
	MasterInternalIP      string `json:"masterInternalIp"`
	ClusterExternalDomain string `json:"clusterExternalDomain"`
	ClusterInternalDomain string `json:"clusterInternalDomain"`

	AllocateNodeCIDRs            bool   `json:"allocateNodeCidrs"`
	EnableClusterMonitoring      string `json:"enableClusterMonitoring"`
	EnableClusterLogging         bool   `json:"enableClusterLogging"`
	EnableNodeLogging            bool   `json:"enableNodeLogging"`
	LoggingDestination           string `json:"loggingDestination"`
	ElasticsearchLoggingReplicas int    `json:"elasticsearchLoggingReplicas"`
	EnableClusterDNS             bool   `json:"enableClusterDns"`
	EnableClusterRegistry        bool   `json:"enableClusterRegistry"`
	ClusterRegistryDisk          string `json:"clusterRegistryDisk"`
	ClusterRegistryDiskSize      string `json:"clusterRegistryDiskSize"`
	DNSReplicas                  int    `json:"dnsReplicas"`
	DNSServerIP                  string `json:"dnsServerIp"`
	DNSDomain                    string `json:"dnsDomain"`
	AdmissionControl             string `json:"admissionControl"`
	MasterIPRange                string `json:"masterIpRange"`
	RuntimeConfig                string `json:"runtimeConfig"`
	StartupConfigToken           string `json:"startupConfigToken"`

	EnableThirdPartyResource bool `json:"enableThirdPartyResource"`

	EnableClusterVPN string `json:"enableClusterVpn"`
	VpnPsk           string `json:"vpnPsk"`

	// ref: https://github.com/appscode/searchlight/blob/master/docs/user-guide/hostfacts/deployment.md
	HostfactsAuthToken string `json:"hostfactsAuthToken"`
	HostfactsCert      string `json:"hostfactsCert"`
	HostfactsKey       string `json:"hostfactsKey"`

	DockerStorage string `json:"dockerStorage"`

	//ClusterName
	//  NodeInstancePrefix
	// Name       string `json:"INSTANCE_PREFIX"`

	// NEW
	NetworkProvider string `json:"networkProvider"` // opencontrail, flannel, kubenet, calico, none
	HairpinMode     string `json:"hairpinMode"`     // promiscuous-bridge, hairpin-veth, none

	EnvTimestamp string `json:"envTimestamp"`

	// TODO: Needed if we build custom Kube image.
	// KubeImageTag       string `json:"KUBE_IMAGE_TAG"`
	KubeDockerRegistry string    `json:"kubeDockerRegistry"`
	Multizone          StrToBool `json:"multizone"`
	NonMasqueradeCidr  string    `json:"nonMasqueradeCidr"`

	KubeletPort                 string `json:"kubeletPort"`
	KubeApiserverRequestTimeout string `json:"kubeApiserverRequestTimeout"`
	TerminatedPodGcThreshold    string `json:"terminatedPodGCThreshold"`
	EnableCustomMetrics         string `json:"enableCustomMetrics"`
	// NEW
	EnableClusterAlert string `json:"enableClusterAlert"`

	Provider string `json:"provider"`
	OS       string `json:"os"`
	Kernel   string `json:"kernel"`

	// Kube 1.3
	// PHID                      string `json:"KUBE_UID"`
	NodeLabels                string `json:"nodeLabels"`
	EnableNodeProblemDetector bool   `json:"enableNodeProblemDetector"`
	EvictionHard              string `json:"evictionHard"`

	ExtraDockerOpts       string `json:"extraDockerOpts"`
	FeatureGates          string `json:"featureGates"`
	NetworkPolicyProvider string `json:"networkPolicyProvider"` // calico

	// Kub1 1.4
	EnableRescheduler bool `json:"enableRescheduler"`

	EnableScheduledJobResource       bool `json:"enableScheduledJobResource"`
	EnableWebhookTokenAuthentication bool `json:"enableWebhookTokenAuthn"`
	EnableWebhookTokenAuthorization  bool `json:"enableWebhookTokenAuthz"`
	EnableRBACAuthorization          bool `json:"enableRbacAuthz"`

	// Cloud Config
	CloudConfigPath  string            `json:"cloudConfig"`
	AzureCloudConfig *AzureCloudConfig `json:"azureCloudConfig"`
	GCECloudConfig   *GCECloudConfig   `json:"gceCloudConfig"`

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
		gen := fireauth.New(os.Getenv("FIREBASE_TOKEN"))
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
	Role               string `json:"role"`
	KubernetesMaster   bool   `json:"kubernetesMaster"`
	InitialEtcdCluster string `json:"initialEtcdCluster"`
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
	SKU              string `json:"sku"`
	Count            int64  `json:"count"`
	UseSpotInstances bool   `json:"useSpotInstances"`
}

type Cluster struct {
	TypeMeta   `json:",inline,omitempty"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       ClusterSpec   `json:"spec,omitempty"`
	Status     ClusterStatus `json:"status,omitempty"`
}

type ClusterSpec struct {
	NodeGroups     []*InstanceGroup `json:"nodeGroups"`
	CredentialName string           `json:"credentialName"`
	KubeadmToken   string           `json:"kubeadmToken"`

	KubernetesVersion string `json:"kubernetesVersion"`

	SSHKeyPHID       string `json:"sshKeyPHID"`
	SSHKeyExternalID string `json:"sshKeyExternalID"`

	// https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/util.sh#L538
	CACertName           string `json:"caCertPHID"`
	FrontProxyCACertName string `json:"frontProxyCaCertPHID"`
	AdminUserCertName    string `json:"userCertPHID"`

	KubeEnv
	// request data. This is needed to give consistent access to these values for all commands.
	Region             string `json:"region"`
	MasterSKU          string `json:"masterSku"`
	DoNotDelete        bool   `json:"-"`
	DefaultAccessLevel string `json:"-"`

	// config
	// Some of these parameters might be useful to expose to users to configure as they please.
	// For now, use the default value used by the Kubernetes project as the default value.

	// TODO: Download the kube binaries from GCS bucket and ignore EU data locality issues for now.

	// common

	// the master root ebs volume size (typically does not need to be very large)
	MasterDiskType string `json:"masterDiskType"`
	MasterDiskSize int64  `json:"masterDiskSize"`
	MasterDiskId   string `json:"masterDiskID"`

	// the node root ebs volume size (used to house docker images)
	NodeDiskType string `json:"nodeDiskType"`
	NodeDiskSize int64  `json:"nodeDiskSize"`

	// GCE: Use Root Field for this in GCE

	// MASTER_TAG="clusterName-master"
	// NODE_TAG="clusterName-node"

	// aws
	// NODE_SCOPES=""

	// gce
	// NODE_SCOPES="${NODE_SCOPES:-compute-rw,monitoring,logging-write,storage-ro}"
	NodeScopes        []string `json:"nodeScopes"`
	PollSleepInterval int      `json:"pollSleepInterval"`

	// If set to Elasticsearch IP, master instance will be associated with this IP.
	// If set to auto, a new Elasticsearch IP will be acquired
	// Otherwise amazon-given public ip will be used (it'll change with reboot).
	MasterReservedIP string `json:"masterReservedIp"`
	MasterExternalIP string `json:"masterExternalIp"`

	// NEW
	// enable various v1beta1 features

	EnableNodePublicIP bool `json:"enableNodePublicIp"`

	EnableNodeAutoscaler  bool    `json:"enableNodeAutoscaler"`
	AutoscalerMinNodes    int     `json:"autoscalerMinNodes"`
	AutoscalerMaxNodes    int     `json:"autoscalerMaxNodes"`
	TargetNodeUtilization float64 `json:"targetNodeUtilization"`

	// instance means either master or node
	InstanceImage        string `json:"instanceImage"`
	InstanceImageProject string `json:"instanceImageProject"`

	// Generated data, always different or every cluster.

	ContainerSubnet string `json:"containerSubnet"` // TODO:where used?

	// only aws

	// Dynamically generated SSH key used for this cluster
	SSHKey *ssh.SSHKey `json:"-"`

	// aws:TAG KubernetesCluster => clusterid
	IAMProfileMaster string `json:"iamProfileMaster"`
	IAMProfileNode   string `json:"iamProfileNode"`
	MasterSGId       string `json:"masterSGID"`
	MasterSGName     string `json:"masterSGName"`
	NodeSGId         string `json:"nodeSGID"`
	NodeSGName       string `json:"nodeSGName"`

	VpcId          string `json:"vpcID"`
	VpcCidr        string `json:"vpcCIDR"`
	VpcCidrBase    string `json:"vpcCIDRBase"`
	MasterIPSuffix string `json:"masterIPSuffix"`
	SubnetId       string `json:"subnetID"`
	SubnetCidr     string `json:"subnetCidr"`
	RouteTableId   string `json:"routeTableID"`
	IGWId          string `json:"igwID"`
	DHCPOptionsId  string `json:"dhcpOptionsID"`

	// only GCE
	Project string `json:"gceProject"`

	// only aws
	RootDeviceName string `json:"-"`

	//only Azure
	InstanceImageVersion    string `json:"instanceImageVersion"`
	AzureStorageAccountName string `json:"azureStorageAccountName"`

	// only Linode
	InstanceRootPassword string `json:"instanceRootPassword"`
}

type ClusterStatus struct {
	Phase  string `json:"phase,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (cluster *Cluster) SetNodeGroups(ng []*proto.InstanceGroup) {
	cluster.Spec.NodeGroups = make([]*InstanceGroup, len(ng))
	for i, g := range ng {
		cluster.Spec.NodeGroups[i] = &InstanceGroup{
			SKU:              g.Sku,
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
func (cluster *Cluster) ApiServerURL() string {
	//if ctx.ApiServerUrl == "" {
	//	host := ctx.Extra().ExternalDomain(ctx.Name)
	//	if ctx.MasterReservedIP != "" {
	//		host = ctx.MasterReservedIP
	//	}
	return fmt.Sprintf("https://%v:6443", cluster.Spec.MasterReservedIP)
	// ctx.Logger().Infoln(fmt.Sprintf("Cluster %v 's api server url: %v\n", ctx.Name, ctx.ApiServerUrl))
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
