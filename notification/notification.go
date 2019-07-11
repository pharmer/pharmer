package notification

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/nats-io/stan.go"
	api "github.com/pharmer/pharmer/apis/v1beta1"
)

type NotificationMessage struct {
	Status  string `json:"status,omitempty"`
	Details string `json:"details,omitempty"`
}

type Notifier struct {
	client  stan.Conn
	subject string
}

var _ api.Logger = &Notifier{}

func (a Notifier) Info(args ...interface{}) {
	_, _ = a.notify(fmt.Sprint(args...))
}

func (a Notifier) Infoln(args ...interface{}) {
	_, _ = a.notify(fmt.Sprintln(args...))
}

func (a Notifier) Infof(format string, args ...interface{}) {
	_, _ = a.notify(fmt.Sprintf(format, args...))
}

func (a Notifier) Debug(args ...interface{}) {
	log.Debugln(args...)
}

func (a Notifier) Debugln(args ...interface{}) {
	log.Debugln(args...)
}

func (a Notifier) Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func (a Notifier) notify(message interface{}) (string, error) {
	var resType struct {
		Fingerprint string `json:"fingerprint"`
	}
	msg := message.(string)
	err := a.client.Publish(a.subject, []byte(msg))
	if err != nil {
		return "", err
	}
	return resType.Fingerprint, err
}

func (a Notifier) Notify(event string, details string) (string, error) {
	return a.notify(details)
}
