package context

import (
	go_ctx "context"

	"github.com/appscode/log"
	"github.com/appscode/pharmer/api"
	"github.com/appscode/pharmer/storage"
)

type Context interface {
	go_ctx.Context

	Store() storage.Storage
	Logger() api.Logger
	Extra() api.DomainManager
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

func NewContext() Context {
	ctx := &FakeContext{
		Context: go_ctx.TODO(),
	}
	ctx.Context = go_ctx.WithValue(ctx.Context, keyExtra, &api.NullDomainManager{})
	ctx.Context = go_ctx.WithValue(ctx.Context, keyLogger, log.New(ctx))
	ctx.Context = go_ctx.WithValue(ctx.Context, keyStore, nil)
	return ctx
}

func (ctx *FakeContext) Store() storage.Storage {
	return ctx.Value(keyStore).(storage.Storage)
}

func (ctx *FakeContext) Logger() api.Logger {
	return ctx.Value(keyLogger).(api.Logger)
}

func (ctx *FakeContext) Extra() api.DomainManager {
	return ctx.Value(keyExtra).(api.DomainManager)
}

func (FakeContext) String() string {
	return "[maverick]"
}
