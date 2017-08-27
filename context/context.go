package context

import (
	go_ctx "context"

	"github.com/appscode/go-dns"
	dns_provider "github.com/appscode/go-dns/provider"
	"github.com/appscode/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/storage/providers/vfs"
)

type Context interface {
	go_ctx.Context

	DNSProvider() dns_provider.Provider
	Store() storage.Interface
	Logger() Logger
	Extra() DomainManager
	String() string
}

var (
	KeyDNS    = struct{}{}
	KeyExtra  = struct{}{}
	KeyLogger = struct{}{}
	KeyStore  = struct{}{}
)

type defaultContext struct {
	go_ctx.Context
}

var _ Context = &defaultContext{}

func (ctx *defaultContext) DNSProvider() dns_provider.Provider {
	return ctx.Value(KeyDNS).(dns_provider.Provider)
}

func (ctx *defaultContext) Store() storage.Interface {
	return ctx.Value(KeyStore).(storage.Interface)
}

func (ctx *defaultContext) Logger() Logger {
	return ctx.Value(KeyLogger).(Logger)
}

func (ctx *defaultContext) Extra() DomainManager {
	return ctx.Value(KeyExtra).(DomainManager)
}

func (defaultContext) String() string {
	return "[-]"
}

type Factory interface {
	New(ctx go_ctx.Context) Context
}

type DefaultFactory struct {
	cfg api.PharmerConfig
}

var _ Factory = &DefaultFactory{}

func NewFactory(cfg api.PharmerConfig) Factory {
	return &DefaultFactory{cfg: cfg}
}

func (f DefaultFactory) New(ctx go_ctx.Context) Context {
	c := &defaultContext{Context: ctx}
	c.Context = go_ctx.WithValue(c.Context, KeyExtra, &NullDomainManager{})
	c.Context = go_ctx.WithValue(c.Context, KeyLogger, log.New(c))
	if store, err := storage.GetProvider(vfs.UID, ctx, &f.cfg); err == nil {
		c.Context = go_ctx.WithValue(c.Context, KeyStore, store)
	} else {
		//fp, _ := storage.GetProvider(fake.UID, ctx, f.cfg)
		//c.Context = go_ctx.WithValue(c.Context, KeyStore, fp)
	}
	c.Context = go_ctx.WithValue(c.Context, KeyDNS, f.getDNSProvider())
	return c
}

func (f DefaultFactory) getDNSProvider() dns_provider.Provider {
	if cred, err := f.cfg.GetCredential(f.cfg.DNS.CredentialName); err == nil {
		if dp, err := dns.NewDNSProvider(cred.Spec.Provider); err == nil {
			return dp
		}
	}
	return &NullDNSProvider{}
}
