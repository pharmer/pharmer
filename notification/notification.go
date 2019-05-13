package notification

import (
	"fmt"

	"github.com/appscode/go/log"
	stan "github.com/nats-io/stan.go"
	api "github.com/pharmer/pharmer/apis/v1beta1"
	"golang.org/x/net/context"
)

type NotificationMessage struct {
	Status  string `json:"status,omitempty"`
	Details string `json:"details,omitempty"`
}

type Notifier struct {
	ctx     context.Context
	client  stan.Conn
	subject string
}

var _ api.Logger = &Notifier{}

func NewNotifier(ctx context.Context, conn stan.Conn, subject string) api.Logger {
	return Notifier{ctx: ctx, client: conn, subject: subject}
}

func (a Notifier) Info(args ...interface{}) {
	a.notify(api.JobStatus_Running, fmt.Sprint(args))
}

func (a Notifier) Infoln(args ...interface{}) {
	a.notify(api.JobStatus_Running, fmt.Sprintln(args))
}

func (a Notifier) Infof(format string, args ...interface{}) {
	a.notify(api.JobStatus_Running, fmt.Sprintf(format, args))
}

func (a Notifier) Debug(args ...interface{}) {
	log.Debugln(args)
}

func (a Notifier) Debugln(args ...interface{}) {
	log.Debugln(args)
}

func (a Notifier) Debugf(format string, args ...interface{}) {
	log.Debugf(format, args)
}

func (a Notifier) notify(event string, message interface{}) (string, error) {
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
	return a.notify(event, details)
}

func (a Notifier) getPath() string {
	return ""
	//return fmt.Sprintf("http://%v?namespace=%v&instance=%v", a.aphlictAdminServerAddr, a.Auth.Username, a.Phid)
}

func (a Notifier) makeRequestBody(event string, message interface{}) interface{} {
	return struct {
		Subscribers []string    `json:"subscribers"`
		Message     interface{} `json:"message,omitempty"`
	}{
		Subscribers: []string{event},
		Message:     message,
	}
}
