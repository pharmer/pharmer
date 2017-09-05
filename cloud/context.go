package cloud

import (
	"context"
	"crypto/rsa"
	"crypto/x509"

	proto "github.com/appscode/api/ssh/v1beta1"
	"github.com/appscode/go-dns"
	dns_provider "github.com/appscode/go-dns/provider"
	"github.com/appscode/go/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/store"
	"github.com/appscode/pharmer/store/providers/fake"
	"github.com/appscode/pharmer/store/providers/vfs"
)

type paramDNS struct{}
type paramExtra struct{}
type paramLogger struct{}
type paramStore struct{}

type paramCACert struct{}
type paramCAKey struct{}
type paramFrontProxyCACert struct{}
type paramFrontProxyCAKey struct{}

type paramSSHKey struct{}

func DNSProvider(ctx context.Context) dns_provider.Provider {
	return ctx.Value(paramDNS{}).(dns_provider.Provider)
}

func Store(ctx context.Context) store.Interface {
	return ctx.Value(paramStore{}).(store.Interface)
}

func Logger(ctx context.Context) api.Logger {
	return ctx.Value(paramLogger{}).(api.Logger)
}

func Extra(ctx context.Context) api.DomainManager {
	return ctx.Value(paramExtra{}).(api.DomainManager)
}

func CACert(ctx context.Context) *x509.Certificate {
	return ctx.Value(paramCACert{}).(*x509.Certificate)
}
func CAKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(paramCAKey{}).(*rsa.PrivateKey)
}

func FrontProxyCACert(ctx context.Context) *x509.Certificate {
	return ctx.Value(paramFrontProxyCACert{}).(*x509.Certificate)
}
func FrontProxyCAKey(ctx context.Context) *rsa.PrivateKey {
	return ctx.Value(paramFrontProxyCAKey{}).(*rsa.PrivateKey)
}

func SSHKey(ctx context.Context) *proto.SSHKey {
	return ctx.Value(paramSSHKey{}).(*proto.SSHKey)
}

func NewContext(parent context.Context, cfg *api.PharmerConfig) context.Context {
	c := parent
	c = context.WithValue(c, paramExtra{}, &api.FakeDomainManager{})
	c = context.WithValue(c, paramLogger{}, log.New(c))
	c = context.WithValue(c, paramStore{}, NewStoreProvider(parent, cfg))
	c = context.WithValue(c, paramDNS{}, NewDNSProvider(cfg))
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
