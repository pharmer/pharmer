package contexts

import (
	"github.com/appscode/log"
	"github.com/appscode/pharmer/storage"
)

type context struct {
	Store  storage.Storage
	Logger Logger
	Extra  DomainInfo
}

// string, bool
type DomainInfo interface {
	Domain(cluster string) string
	ExternalDomain(cluster string) string
	InternalDomain(cluster string) string
}

type Logger interface {
	Info(args ...interface{})
	Infoln(args ...interface{})
	Infof(format string, args ...interface{})

	Debug(args ...interface{})
	Debugln(args ...interface{})
	Debugf(format string, args ...interface{})
}

func NewContext() *context {
	ctx := &context{
		Store: nil,
	}
	ctx.Logger = log.New(ctx)
	return ctx
}

func (c *context) String() string {
	return "[maverick]"
}
