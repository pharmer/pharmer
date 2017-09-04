package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"

	"github.com/appscode/go-dns"
	dns_provider "github.com/appscode/go-dns/provider"
	"github.com/appscode/go/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/store"
	"github.com/appscode/pharmer/store/providers/fake"
	"github.com/appscode/pharmer/store/providers/vfs"
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

func Store(ctx context.Context) store.Interface {
	return ctx.Value(keyStore{}).(store.Interface)
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

func NewStoreProvider(ctx context.Context, cfg *api.PharmerConfig) store.Interface {
	if store, err := store.GetProvider(vfs.UID, ctx, cfg); err == nil {
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
