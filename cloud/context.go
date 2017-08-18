package cloud

import (
	"github.com/appscode/errors"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/context"
)

// This is any provider != aws, azure, gce
func LoadDefaultGenericContext(ctx context.Context, cluster *api.Cluster) error {
	err := cluster.Spec.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	cluster.Spec.ClusterExternalDomain = ctx.Extra().ExternalDomain(cluster.Name)
	cluster.Spec.ClusterInternalDomain = ctx.Extra().InternalDomain(cluster.Name)

	cluster.Status.Phase = api.ClusterPhasePending
	cluster.Spec.OS = "debian"

	//-------------------------- ctx.MasterSKU = "94" // 2 cpu
	cluster.Spec.DockerStorage = "aufs"

	// Using custom image with memory controller enabled
	// -------------------------ctx.InstanceImage = "16604964" // "container-os-20160402" // Debian 8.4 x64

	cluster.Spec.MasterReservedIP = "" // "auto"
	cluster.Spec.MasterIPRange = "10.246.0.0/24"
	cluster.Spec.ClusterIPRange = "10.244.0.0/16"
	cluster.Spec.ServiceClusterIPRange = "10.0.0.0/16"
	cluster.Spec.NodeScopes = []string{}
	cluster.Spec.PollSleepInterval = 3

	cluster.Spec.RegisterMasterKubelet = true
	cluster.Spec.EnableNodePublicIP = true

	// Kubelet is responsible for cidr allocation via cni plugin
	cluster.Spec.AllocateNodeCIDRs = true

	cluster.Spec.EnableClusterMonitoring = "appscode"
	cluster.Spec.EnableNodeLogging = true
	cluster.Spec.LoggingDestination = "appscode-elasticsearch"
	cluster.Spec.EnableClusterLogging = true
	cluster.Spec.ElasticsearchLoggingReplicas = 1

	cluster.Spec.ExtraDockerOpts = ""

	cluster.Spec.EnableClusterDNS = true
	cluster.Spec.DNSServerIP = "10.0.0.10"
	cluster.Spec.DNSDomain = "cluster.Spec.local"
	cluster.Spec.DNSReplicas = 1

	// TODO: Needs multiple auto scaler
	cluster.Spec.EnableNodeAutoscaler = false

	cluster.Spec.AdmissionControl = "NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota"

	cluster.Spec.NetworkProvider = "kube-flannel"
	cluster.Spec.HairpinMode = "promiscuous-bridge"
	// ctx.NonMasqueradeCidr = "10.0.0.0/8"
	// ctx.EnableDnssyncer = true

	cluster.Spec.EnableClusterVPN = "h2h-psk"
	cluster.Spec.VpnPsk = rand.GeneratePassword()

	BuildRuntimeConfig(cluster)
	return nil
}

func NewInstances(ctx context.Context, cluster *api.Cluster) (*api.ClusterInstances, error) {
	p, err := GetProvider("", nil) // TODO: FixIt!
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New(cluster.Spec.Provider + " is an unknown Kubernetes cloud.").WithContext(ctx).Err()
	}
	return cluster.NewInstances(p.MatchInstance)
}
