package contexts

import (
	"github.com/appscode/log"
	"github.com/appscode/pharmer/storage"
)

type context struct {
	Store  storage.Storage
	logger Logger
}

type Logger interface {
	Info(args ...interface{})
	Infoln(args ...interface{})
	Infof(format string, args ...interface{})

	Debug(args ...interface{})
	Debugln(args ...interface{})
	Debugf(format string, args ...interface{})
}

func (c *context) Logger() Logger {
	return c.logger
}

func NewContext() *context {
	ctx := &context{
		Store: nil,
	}
	ctx.logger = log.New(ctx)
	return ctx
}

func (c *context) String() string {
	return "[self-hosted]"
}
