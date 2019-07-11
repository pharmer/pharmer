package ulogr

import (
	"github.com/go-logr/logr"
)

type unionLogger []logr.Logger
type unionInfoLogger []logr.InfoLogger

var _ logr.Logger = unionLogger{}

func NewLogger(loggers ...logr.Logger) logr.Logger {
	return unionLogger(loggers)
}

func (l unionLogger) Info(msg string, keysAndValues ...interface{}) {
	for _, logger := range l {
		if logger.Enabled() {
			logger.Info(msg, keysAndValues...)
		}
	}
}

func (il unionInfoLogger) Info(msg string, keysAndValues ...interface{}) {
	for _, logger := range il {
		if logger.Enabled() {
			logger.Info(msg, keysAndValues...)
		}
	}
}

func (l unionLogger) Enabled() bool {
	enabled := false
	for _, logger := range l {
		enabled = enabled || logger.Enabled()
	}

	return enabled
}

func (il unionInfoLogger) Enabled() bool {
	for _, logger := range il {
		if logger.Enabled() {
			return true
		}
	}

	return false
}

func (l unionLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	for _, logger := range l {
		logger.Error(err, msg, keysAndValues...)
	}
}

func (l unionLogger) V(level int) logr.InfoLogger {
	out := make([]logr.InfoLogger, 0, len(l))
	for _, logger := range l {
		out = append(out, logger.V(level))
	}

	return unionInfoLogger(out)
}

func (l unionLogger) WithName(name string) logr.Logger {
	out := make([]logr.Logger, 0, len(l))
	for _, logger := range l {
		out = append(out, logger.WithName(name))
	}
	return unionLogger(out)
}

func (l unionLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	out := make([]logr.Logger, 0, len(l))
	for _, logger := range l {
		out = append(out, logger.WithValues(keysAndValues...))
	}
	return unionLogger(out)
}
