package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"

	"github.com/appscode/go-dns"
	dns_provider "github.com/appscode/go-dns/provider"
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/errors"
	"github.com/appscode/go/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/storage/providers/fake"
	"github.com/appscode/pharmer/storage/providers/vfs"
)

type keyDNS struct{}
type keyExtra struct{}
type keyLogger struct{}
type keyStore struct{}

type keyCACert struct{}
type keyCAKey struct{}
type keyFrontProxyCACert struct{}
type keyFrontProxyCAKey struct{}
type keyAdminUserCert struct{}
type keyAdminUserKey struct{}

func DNSProvider(ctx context.Context) dns_provider.Provider {
	return ctx.Value(keyDNS{}).(dns_provider.Provider)
}

func Store(ctx context.Context) storage.Interface {
	return ctx.Value(keyStore{}).(storage.Interface)
}

func Logger(ctx context.Context) api.Logger {
	return ctx.Value(keyLogger{}).(api.Logger)
}

func Extra(ctx context.Context) api.DomainManager {
	return ctx.Value(keyExtra{}).(api.DomainManager)
}

func CACert(ctx context.Context) *x509.Certificate {
	return ctx.Value(keyCACert{}).(*x509.Certificate)
}
func CAKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(keyCAKey{}).(*rsa.PrivateKey)
}

func FrontProxyCACert(ctx context.Context) *x509.Certificate {
	return ctx.Value(keyFrontProxyCACert{}).(*x509.Certificate)
}
func FrontProxyCAKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(keyFrontProxyCAKey{}).(*rsa.PrivateKey)
}

func AdminUserCert(ctx context.Context) *x509.Certificate {
	return ctx.Value(keyAdminUserCert{}).(*x509.Certificate)
}
func AdminUserKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(keyAdminUserKey{}).(*rsa.PrivateKey)
}

func NewContext(parent context.Context, cfg *api.PharmerConfig) context.Context {
	c := parent
	c = context.WithValue(c, keyExtra{}, &api.FakeDomainManager{})
	c = context.WithValue(c, keyLogger{}, log.New(c))
	c = context.WithValue(c, keyStore{}, NewStoreProvider(parent, cfg))
	c = context.WithValue(c, keyDNS{}, NewDNSProvider(cfg))
	return c
}

func NewStoreProvider(ctx context.Context, cfg *api.PharmerConfig) storage.Interface {
	if store, err := storage.GetProvider(vfs.UID, ctx, cfg); err == nil {
		return store
	}
	return &fake.FakeStore{}
}

func NewDNSProvider(cfg *api.PharmerConfig) dns_provider.Provider {
	if cfg.DNS != nil {
		if cred, err := cfg.GetCredential(cfg.DNS.CredentialName); err == nil {
			if dp, err := dns.NewDNSProvider(cred.Spec.Provider); err == nil {
				return dp
			}
		}
	}
	return &api.FakeDNSProvider{}
}

// This is any provider != aws, azure, gce
func LoadDefaultGenericContext(ctx context.Context, cluster *api.Cluster) error {
	err := cluster.Spec.KubeEnv.SetDefaults()
	if err != nil {
		return errors.FromErr(err).WithContext(ctx).Err()
	}

	cluster.Spec.ClusterExternalDomain = Extra(ctx).ExternalDomain(cluster.Name)
	cluster.Spec.ClusterInternalDomain = Extra(ctx).InternalDomain(cluster.Name)

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
	//p, err := GetProvider("", nil) // TODO: FixIt!
	//if err != nil {
	//	return nil, err
	//}
	//if p == nil {
	//	return nil, errors.New(cluster.Spec.Provider + " is an unknown Kubernetes cloud.").WithContext(ctx).Err()
	//}
	return cluster.NewInstances(nil)
}
