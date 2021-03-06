//go:generate mockery -inpkg -name=SendGridClient

package utils

import (
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendGridClient interface {
	Send(email *mail.SGMailV3) (*rest.Response, error)
}
