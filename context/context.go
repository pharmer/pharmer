package context

import (
	go_ctx "context"
	"fmt"

	"github.com/appscode/go-dns/aws"
	"github.com/appscode/go-dns/azure"
	"github.com/appscode/go-dns/cloudflare"
	"github.com/appscode/go-dns/digitalocean"
	"github.com/appscode/go-dns/googlecloud"
	"github.com/appscode/go-dns/linode"
	dns "github.com/appscode/go-dns/provider"
	"github.com/appscode/go-dns/vultr"
	"github.com/appscode/log"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/storage"
)

type Context interface {
	go_ctx.Context

	DNSProvider() dns.Provider
	Store() storage.Storage
	Logger() Logger
	Extra() DomainManager
	String() string
}

type FakeContext struct {
	go_ctx.Context
}

var (
	keyDNS    = struct{}{}
	keyExtra  = struct{}{}
	keyLogger = struct{}{}
	keyStore  = struct{}{}
)

var _ Context = &FakeContext{}

func NewContext(ctx go_ctx.Context, cfg *config.PharmerConfig) Context {
	c := &FakeContext{Context: ctx}
	c.Context = go_ctx.WithValue(c.Context, keyExtra, &NullDomainManager{})
	c.Context = go_ctx.WithValue(c.Context, keyLogger, log.New(c))
	c.Context = go_ctx.WithValue(c.Context, keyStore, nil)
	if dp, err := newDNSProvider(cfg); err == nil {
		c.Context = go_ctx.WithValue(c.Context, keyDNS, dp)
	}
	return c
}

func newDNSProvider(cfg *config.PharmerConfig) (dns.Provider, error) {
	curCtx := cfg.Context("")
	switch curCtx.DNS.Provider {
	case "azure":
		return azure.NewDNSProviderCredentials(curCtx.DNS.Azure)
	case "cloudflare":
		return cloudflare.NewDNSProviderCredentials(curCtx.DNS.Cloudflare)
	case "digitalocean":
		return digitalocean.NewDNSProviderCredentials(curCtx.DNS.Digitalocean)
	case "gcloud":
		return googlecloud.NewDNSProviderCredentials(curCtx.DNS.Gcloud)
	case "linode":
		return linode.NewDNSProviderCredentials(curCtx.DNS.Linode)
	case "aws":
	case "route53":
		return aws.NewDNSProviderCredentials(curCtx.DNS.AWS)
	case "vultr":
		return vultr.NewDNSProviderCredentials(curCtx.DNS.Vultr)
	}
	return nil, fmt.Errorf("Unrecognised DNS provider: %s", curCtx.DNS.Provider)
}

func (ctx *FakeContext) DNSProvider() dns.Provider {
	return ctx.Value(keyDNS).(dns.Provider)
}

func (ctx *FakeContext) Store() storage.Storage {
	return ctx.Value(keyStore).(storage.Storage)
}

func (ctx *FakeContext) Logger() Logger {
	return ctx.Value(keyLogger).(Logger)
}

func (ctx *FakeContext) Extra() DomainManager {
	return ctx.Value(keyExtra).(DomainManager)
}

func (FakeContext) String() string {
	return "[maverick]"
}
