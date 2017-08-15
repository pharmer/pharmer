package context

import (
	go_ctx "context"

	"github.com/appscode/log"
	"github.com/appscode/pharmer/config"
	"github.com/appscode/pharmer/storage"
)

type Context interface {
	go_ctx.Context

	Store() storage.Storage
	Logger() Logger
	Extra() DomainManager
	String() string
}

type FakeContext struct {
	go_ctx.Context
}

var (
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
	return c
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
