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
	Store() storage.Store
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

func (ctx *defaultContext) Store() storage.Store {
	return ctx.Value(KeyStore).(storage.Store)
}

func (ctx *defaultContext) Logger() Logger {
	return ctx.Value(KeyLogger).(Logger)
}

func (ctx *defaultContext) Extra() DomainManager {
	return ctx.Value(KeyExtra).(DomainManager)
}

func (defaultContext) String() string {
	return "[maverick]"
}

type Factory interface {
	New(ctx go_ctx.Context) Context
}

type DefaultFactory struct {
	Config config.PharmerConfig
}

var _ Factory = &DefaultFactory{}

func (f DefaultFactory) New(ctx go_ctx.Context) Context {
	c := &defaultContext{Context: ctx}
	c.Context = go_ctx.WithValue(c.Context, KeyExtra, &NullDomainManager{})
	c.Context = go_ctx.WithValue(c.Context, KeyLogger, log.New(c))
	c.Context = go_ctx.WithValue(c.Context, KeyStore, fake.FakeStore{Config: &f.Config})
	//if dp, err := newDNSProvider(cfg); err == nil {
	//	c.Context = go_ctx.WithValue(c.Context, KeyDNS, dp)
	//}
	return c
}

func newDNSProvider(cfg *config.PharmerConfig) (dns.Provider, error) {
	//curCtx := cfg.Context("")
	//switch curCtx.DNS.Provider {
	//case "azure":
	//	return azure.NewDNSProviderCredentials(curCtx.DNS.Azure)
	//case "cloudflare":
	//	return cloudflare.NewDNSProviderCredentials(curCtx.DNS.Cloudflare)
	//case "digitalocean":
	//	return digitalocean.NewDNSProviderCredentials(curCtx.DNS.Digitalocean)
	//case "gcloud":
	//	return googlecloud.NewDNSProviderCredentials(curCtx.DNS.Gcloud)
	//case "linode":
	//	return linode.NewDNSProviderCredentials(curCtx.DNS.Linode)
	//case "aws":
	//case "route53":
	//	return aws.NewDNSProviderCredentials(curCtx.DNS.AWS)
	//case "vultr":
	//	return vultr.NewDNSProviderCredentials(curCtx.DNS.Vultr)
	//}
	return nil, nil // fmt.Errorf("Unrecognised DNS provider: %s", curCtx.DNS.Provider)
}
