package errorhandlers

import (
	"github.com/appscode/errors"
	"github.com/appscode/errors/h/log"
)

func init() {
	errors.Handlers.Add(log.LogHandler{})
}
