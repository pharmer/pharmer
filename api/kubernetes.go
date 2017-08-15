package api

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	proto "github.com/appscode/api/kubernetes/v1beta1"
	ssh "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	_env "github.com/appscode/go/env"
	"github.com/golang/protobuf/jsonpb"
	"k8s.io/client-go/rest"
)

type InstanceGroup struct {
	Sku              string `json:"SKU"`
	Count            int64  `json:"COUNT"`
	UseSpotInstances bool   `json:"USE_SPOT_INSTANCES"`
}

// Embed this context in actual providers.
type Cluster struct {
	KubeEnv
	CommonNonEnv

	// request data. This is needed to give consistent access to these values for all commands.
	Region              string            `json:"REGION"`
	MasterSKU           string            `json:"MASTER_SKU"`
	NodeSet             map[string]int64  `json:"NODE_SET"` // deprecated, use NODES
	NodeGroups          []*InstanceGroup  `json:"NODE_GROUPS"`
	CloudCredentialPHID string            `json:"CLOUD_CREDENTIAL_PHID"`
	CloudCredential     map[string]string `json:"-"`
	Status              string            `json:"-"`
	StatusCause         string            `json:"-"`
	DoNotDelete         bool              `json:"-"`
	DefaultAccessLevel  string            `json:"-"`

	KubeVersion        string `json:"KUBE_VERSION"`
	KubeServerVersion  string `json:"KUBE_SERVER_VERSION"`
	SaltbaseVersion    string `json:"SALTBASE_VERSION"`
	KubeStarterVersion string `json:"KUBE_STARTER_VERSION"`
	HostfactsVersion   string `json:"HOSTFACTS_VERSION"`

	AppsCodeLogIndexPrefix            string `json:"APPSCODE_LOG_INDEX_PREFIX"`
	AppsCodeLogStorageLifetime        int64  `json:"APPSCODE_LOG_STORAGE_LIFETIME"`
	AppsCodeMonitoringStorageLifetime int64  `json:"APPSCODE_MONITORING_STORAGE_LIFETIME"`

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
	CaCertPHID            string `json:"CA_CERT_PHID"`
	MasterCertPHID        string `json:"MASTER_CERT_PHID"`
	DefaultLBCertPHID     string `json:"DEFAULT_LB_CERT_PHID"`
	KubeletCertPHID       string `json:"KUBELET_CERT_PHID"`
	KubeAPIServerCertPHID string `json:"KUBE_API_SERVER_CERT_PHID"`
	HostfactsCertPHID     string `json:"HOSTFACTS_CERT_PHID"`

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

func (ctx *Cluster) SetNodeGroups(ng []*proto.InstanceGroup) {
	ctx.NodeGroups = make([]*InstanceGroup, len(ng))
	for i, g := range ng {
		ctx.NodeGroups[i] = &InstanceGroup{
			Sku:              g.Sku,
			Count:            g.Count,
			UseSpotInstances: g.UseSpotInstances,
		}
	}
}

func (ctx *Cluster) Save() error {
	return nil
}

func (ctx *Cluster) AddEdge(src, dst string, typ ClusterOP) error {
	return nil
}

// Set ctx.Name (required)
// Set ctx.ContextVersion (optional) to load specific version
func (ctx *Cluster) Load() error {
	return nil
}

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

func (ctx *Cluster) Delete() error {
	if ctx.Status == KubernetesStatus_Pending || ctx.Status == KubernetesStatus_Failing || ctx.Status == KubernetesStatus_Failed {
		ctx.Status = KubernetesStatus_Failed
	} else {
		ctx.Status = KubernetesStatus_Deleted
	}
	if err := ctx.Save(); err != nil {
		return err
	}

	n := rand.WithUniqSuffix(ctx.Name)
	//if _, err := ctx.Store().Engine.Update(&Kubernetes{Name: n}, &Kubernetes{PHID: ctx.PHID}); err != nil {
	//	return err
	//}
	ctx.Name = n
	return nil
}

func (ctx *Cluster) clusterIP(seq int64) string {
	octets := strings.Split(ctx.ServiceClusterIPRange, ".")
	p, _ := strconv.ParseInt(octets[3], 10, 64)
	p = p + seq
	octets[3] = strconv.FormatInt(p, 10)
	return strings.Join(octets, ".")
}

func (ctx *Cluster) KubernetesClusterIP() string {
	return ctx.clusterIP(1)
}

// This is a onetime initializer method.
func (ctx *Cluster) DetectApiServerURL() {
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

func (ctx *Cluster) NodeCount() int64 {
	n := int64(0)
	if ctx.RegisterMasterKubelet {
		n = 1
	}
	for _, ng := range ctx.NodeGroups {
		n += ng.Count
	}
	return n
}

func (ctx *Cluster) StartupConfig(role string) *ClusterStartupConfig {
	var config ClusterStartupConfig
	config.KubeEnv = ctx.KubeEnv
	config.CommonNonEnv = ctx.CommonNonEnv
	config.Role = role
	config.KubernetesMaster = role == RoleKubernetesMaster
	config.InitialEtcdCluster = ctx.KubernetesMasterName
	config.NumNodes = ctx.NodeCount()
	return &config
}

func (ctx *Cluster) StartupConfigJson(role string) (string, error) {
	confJson, err := json.Marshal(ctx.StartupConfig(role))
	if err != nil {
		return "", err
	}
	return string(confJson), nil
}

func (ctx *Cluster) StartupConfigResponse(role string) (string, error) {
	confJson, err := ctx.StartupConfigJson(role)
	if err != nil {
		return "", err
	}

	resp := &proto.ClusterStartupConfigResponse{
		Configuration: string(confJson),
	}
	m := jsonpb.Marshaler{}
	return m.MarshalToString(resp)
}

// WARNING:
// Returned KubeClient uses admin bearer token. This should only be used for cluster provisioning operations.
// For other cluster operations initiated by users, use KubeAddon context.
func (ctx *Cluster) NewKubeClient() (*kubeClient, error) {
	kubeconfig := &rest.Config{
		Host:        ctx.ApiServerUrl,
		BearerToken: ctx.KubeBearerToken,
	}
	if _env.FromHost().DevMode() {
		kubeconfig.Insecure = true
	} else {
		caCert, err := base64.StdEncoding.DecodeString(ctx.CaCert)
		if err != nil {
			return nil, err
		}
		kubeconfig.TLSClientConfig = rest.TLSClientConfig{
			CAData: caCert,
		}
	}
	return NewKubeClient(kubeconfig)
}

func (ctx *Cluster) NewInstances(matches func(i *KubernetesInstance, md *InstanceMetadata) bool) (*ClusterInstances, error) {
	if matches == nil {
		return nil, errors.New(`Use "github.com/appscode/pharmer/cloud/lib".NewInstances`).Err()
	}
	return &ClusterInstances{
		matches:        matches,
		KubernetesPHID: ctx.PHID,
		Instances:      make([]*KubernetesInstance, 0),
	}, nil
}
