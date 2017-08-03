package errorhandlers

import (
	"sync"

	"github.com/appscode/errors"
	"github.com/appscode/errors/h/mail"
	"github.com/appscode/go-notify"
	"github.com/appscode/go-notify/mailgun"
	"github.com/appscode/go-notify/smtp"
	_env "github.com/appscode/go/env"
	"github.com/appscode/pharmer/system"
)

const (
	MailFrom     = "postmaster@"
	MailToSuffix = "@appscode.com"
)

var cOnce sync.Once

func Init() {
	cOnce.Do(func() {
		// initialize the error handlers in sequence
		if !_env.FromHost().DevMode() {
			system.Init()
			if h := NewEmailHandler(); h != nil {
				errors.Handlers.Add(h)
			}
		}
	})
}

func NewEmailHandler() *mail.EmailHandler {
	var mailer notify.ByEmail
	if system.Config.Mail.Mailer == mailgun.UID {
		mailer = mailgun.New(mailgun.Options{
			Domain: system.Config.Mail.PublicDomain,
			ApiKey: system.Config.Mail.Mailgun.ApiKey,
		})
	} else if system.Config.Mail.Mailer == smtp.UID {
		mailer = smtp.New(system.Config.Mail.SMTP)
	} else {
		return nil
	}
	mailer = mailer.
		From(MailFrom + system.PublicBaseDomain()).
		To("oplog" + "+" + "api" + "-" + _env.FromHost().String() + MailToSuffix)
	return mail.NewEmailhandler(mailer)
}

func SendMailAndIgnore(err error) {
	if err != nil {
		errors.FromErr(err).Err()
	}
}

func SendMailWithContextAndIgnore(ctx errors.Context, err error) {
	if err != nil {
		errors.FromErr(err).WithContext(ctx).Err()
	}
}
