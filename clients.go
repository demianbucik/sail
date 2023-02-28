//go:generate mockery --inpackage --name=SendGridClient
//go:generate mockery --inpackage --name=ReCaptchaClient

package sail

import (
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gopkg.in/ezzarghili/recaptcha-go.v4"
)

type SendGridClient interface {
	Send(email *mail.SGMailV3) (*rest.Response, error)
}

type ReCaptchaClient interface {
	Verify(challenge string) error
	VerifyWithOptions(challenge string, options recaptcha.VerifyOption) error
}
