package context

import "github.com/appscode/log"

type Logger interface {
	Info(args ...interface{})
	Infoln(args ...interface{})
	Infof(format string, args ...interface{})

	Debug(args ...interface{})
	Debugln(args ...interface{})
	Debugf(format string, args ...interface{})
}

var _ Logger = log.New(nil)
