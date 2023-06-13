//go:generate mockery --inpackage --name=SendGridClient
//go:generate mockery --inpackage --name=ReCaptchaClient

package sail

import (
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/demianbucik/sail/utils"
)

type SendGridClient interface {
	Send(email *mail.SGMailV3) (*rest.Response, error)
}

type ReCaptchaClient interface {
	Verify(response string, opts utils.VerifyOptions) error
}
