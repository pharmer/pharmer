package lib

import (
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/storage"
)

// This is any provider != aws, azure, gce
func LoadDefaultGenericContext(ctx *api.Cluster) error {
	err := ctx.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	ctx.ClusterExternalDomain = ctx.Extra().ExternalDomain(ctx.Name)
	ctx.ClusterInternalDomain = ctx.Extra().InternalDomain(ctx.Name)

	ctx.Status = storage.KubernetesStatus_Pending
	ctx.OS = "debian"

	ctx.AppsCodeLogIndexPrefix = "logstash-"
	ctx.AppsCodeLogStorageLifetime = 90 * 24 * 3600
	ctx.AppsCodeMonitoringStorageLifetime = 90 * 24 * 3600

	//-------------------------- ctx.MasterSKU = "94" // 2 cpu
	ctx.DockerStorage = "aufs"

	// Using custom image with memory controller enabled
	// -------------------------ctx.InstanceImage = "16604964" // "container-os-20160402" // Debian 8.4 x64

	ctx.MasterReservedIP = "" // "auto"
	ctx.MasterIPRange = "10.246.0.0/24"
	ctx.ClusterIPRange = "10.244.0.0/16"
	ctx.ServiceClusterIPRange = "10.0.0.0/16"
	ctx.NodeScopes = []string{}
	ctx.PollSleepInterval = 3

	ctx.RegisterMasterKubelet = true
	ctx.EnableNodePublicIP = true

	// Kubelet is responsible for cidr allocation via cni plugin
	ctx.AllocateNodeCIDRs = true

	ctx.EnableClusterMonitoring = "appscode"
	ctx.EnableNodeLogging = true
	ctx.LoggingDestination = "appscode-elasticsearch"
	ctx.EnableClusterLogging = true
	ctx.ElasticsearchLoggingReplicas = 1

	ctx.ExtraDockerOpts = ""

	ctx.EnableClusterDNS = true
	ctx.DNSServerIP = "10.0.0.10"
	ctx.DNSDomain = "cluster.local"
	ctx.DNSReplicas = 1

	// TODO: Needs multiple auto scaler
	ctx.EnableNodeAutoscaler = false

	ctx.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"

	ctx.NetworkProvider = "kube-flannel"
	ctx.HairpinMode = "promiscuous-bridge"
	// ctx.NonMasqueradeCidr = "10.0.0.0/8"
	// ctx.EnableDnssyncer = true

	ctx.EnableClusterVPN = "h2h-psk"
	ctx.VpnPsk = rand.GeneratePassword()

	BuildRuntimeConfig(ctx)
	return nil
}

func NewInstances(ctx *api.Cluster) (*api.ClusterInstances, error) {
	p := extpoints.Providers.Lookup(ctx.Provider)
	if p == nil {
		return nil, errors.New(ctx.Provider + " is an unknown Kubernetes lib.").WithContext(ctx).Err()
	}
	return ctx.NewInstances(p.MatchInstance)
}
