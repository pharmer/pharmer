package context

import (
	go_ctx "context"

	dns "github.com/appscode/go-dns/provider"
	"github.com/appscode/log"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/storage"
	"github.com/appscode/pharmer/storage/providers/fake"
)

type Context interface {
	go_ctx.Context

	DNSProvider() dns.Provider
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

func (ctx *defaultContext) DNSProvider() dns.Provider {
	return ctx.Value(KeyDNS).(dns.Provider)
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
	cfg config.PharmerConfig
}

var _ Factory = &DefaultFactory{}

func NewFactory(cfg config.PharmerConfig) Factory {
	return &DefaultFactory{cfg: cfg}
}

func (f DefaultFactory) New(ctx go_ctx.Context) Context {
	c := &defaultContext{Context: ctx}
	c.Context = go_ctx.WithValue(c.Context, KeyExtra, &NullDomainManager{})
	c.Context = go_ctx.WithValue(c.Context, KeyLogger, log.New(c))
	if sp, err := storage.GetProvider("", ctx, f.cfg); err == nil {
		c.Context = go_ctx.WithValue(c.Context, KeyStore, sp)
	} else {
		fp, _ := storage.GetProvider(fake.UID, ctx, f.cfg)
		c.Context = go_ctx.WithValue(c.Context, KeyStore, fp)
	}
	c.Context = go_ctx.WithValue(c.Context, KeyDNS, &NullDNSProvider{})
	return c
}
