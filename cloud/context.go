package cloud

import (
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/context"
)

// This is any provider != aws, azure, gce
func LoadDefaultGenericContext(ctx context.Context, cluster *api.Cluster) error {
	err := cluster.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	cluster.ClusterExternalDomain = ctx.Extra().ExternalDomain(cluster.Name)
	cluster.ClusterInternalDomain = ctx.Extra().InternalDomain(cluster.Name)

	cluster.Status = api.KubernetesStatus_Pending
	cluster.OS = "debian"

	//-------------------------- ctx.MasterSKU = "94" // 2 cpu
	cluster.DockerStorage = "aufs"

	// Using custom image with memory controller enabled
	// -------------------------ctx.InstanceImage = "16604964" // "container-os-20160402" // Debian 8.4 x64

	cluster.MasterReservedIP = "" // "auto"
	cluster.MasterIPRange = "10.246.0.0/24"
	cluster.ClusterIPRange = "10.244.0.0/16"
	cluster.ServiceClusterIPRange = "10.0.0.0/16"
	cluster.NodeScopes = []string{}
	cluster.PollSleepInterval = 3

	cluster.RegisterMasterKubelet = true
	cluster.EnableNodePublicIP = true

	// Kubelet is responsible for cidr allocation via cni plugin
	cluster.AllocateNodeCIDRs = true

	cluster.EnableClusterMonitoring = "appscode"
	cluster.EnableNodeLogging = true
	cluster.LoggingDestination = "appscode-elasticsearch"
	cluster.EnableClusterLogging = true
	cluster.ElasticsearchLoggingReplicas = 1

	cluster.ExtraDockerOpts = ""

	cluster.EnableClusterDNS = true
	cluster.DNSServerIP = "10.0.0.10"
	cluster.DNSDomain = "cluster.local"
	cluster.DNSReplicas = 1

	// TODO: Needs multiple auto scaler
	cluster.EnableNodeAutoscaler = false

	cluster.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"

	cluster.NetworkProvider = "kube-flannel"
	cluster.HairpinMode = "promiscuous-bridge"
	// ctx.NonMasqueradeCidr = "10.0.0.0/8"
	// ctx.EnableDnssyncer = true

	cluster.EnableClusterVPN = "h2h-psk"
	cluster.VpnPsk = rand.GeneratePassword()

	BuildRuntimeConfig(cluster)
	return nil
}

func NewInstances(ctx context.Context, cluster *api.Cluster) (*api.ClusterInstances, error) {
	p, err := GetProvider("", nil) // TODO: FixIt!
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New(cluster.Provider + " is an unknown Kubernetes cloud.").WithContext(ctx).Err()
	}
	return cluster.NewInstances(p.MatchInstance)
}
